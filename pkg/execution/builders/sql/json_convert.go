package sql

import (
	"fmt"
	"slices"

	"github.com/doug-martin/goqu/v9/exp"
	"github.com/spf13/cast"
)

// ConvertFilterMapToExpression converts a FilterInput-style map to a JSON filter expression
// This replaces BuildJsonFilterFromOperatorMap with expression-based building
func ConvertFilterMapToExpression(
	col exp.IdentifierExpression,
	filterMap map[string]any,
	dialect Dialect,
) (exp.Expression, error) {
	return convertFilterMapWithPrefix(col, filterMap, "", dialect)
}

// buildNestedConditionString builds a JSONPath condition string from a filter map, handling nested AND/OR
// Returns the condition string, variables map, and variable counter offset
func buildNestedConditionString(
	filterMap map[string]any,
	pathPrefix string,
	varOffset int,
	logic LogicType,
) (string, map[string]any, int, error) {
	vars := make(map[string]any)
	parts := make([]string, 0)
	currentVarOffset := varOffset

	// Sort keys for deterministic output
	keys := make([]string, 0, len(filterMap))
	for k := range filterMap {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	for _, field := range keys {
		opMapRaw := filterMap[field]

		switch field {
		case "AND":
			andFilters, ok := opMapRaw.([]any)
			if !ok {
				return "", nil, 0, fmt.Errorf("AND must be an array")
			}

			andParts := make([]string, 0)
			for _, af := range andFilters {
				afMap, ok := af.(map[string]any)
				if !ok {
					return "", nil, 0, fmt.Errorf("AND element must be a map")
				}

				// Recursively build condition string for this AND element
				condStr, subVars, newOffset, err := buildNestedConditionString(afMap, pathPrefix, currentVarOffset, LogicAnd)
				if err != nil {
					return "", nil, 0, err
				}
				andParts = append(andParts, condStr)
				for k, v := range subVars {
					vars[k] = v
				}
				currentVarOffset = newOffset
			}

			if len(andParts) > 1 {
				// Multiple parts - combine with AND
				combined := ""
				for i, part := range andParts {
					if i > 0 {
						combined += " && "
					}
					// Add parentheses if part contains OR or multiple conditions
					if len(andParts) > 1 && (contains(part, " || ") || (len(andParts) > 1 && i > 0)) {
						combined += fmt.Sprintf("(%s)", part)
					} else {
						combined += part
					}
				}
				parts = append(parts, combined)
			} else if len(andParts) == 1 {
				parts = append(parts, andParts[0])
			}

		case "OR":
			orFilters, ok := opMapRaw.([]any)
			if !ok {
				return "", nil, 0, fmt.Errorf("OR must be an array")
			}

			orParts := make([]string, 0)
			for _, of := range orFilters {
				ofMap, ok := of.(map[string]any)
				if !ok {
					return "", nil, 0, fmt.Errorf("OR element must be a map")
				}

				// Recursively build condition string for this OR element
				condStr, subVars, newOffset, err := buildNestedConditionString(ofMap, pathPrefix, currentVarOffset, LogicOr)
				if err != nil {
					return "", nil, 0, err
				}
				orParts = append(orParts, condStr)
				for k, v := range subVars {
					vars[k] = v
				}
				currentVarOffset = newOffset
			}

			if len(orParts) > 1 {
				// Multiple parts - combine with OR
				combined := ""
				for i, part := range orParts {
					if i > 0 {
						combined += " || "
					}
					combined += part
				}
				// Wrap in parentheses for OR (will get double parentheses when used in top-level OR)
				parts = append(parts, fmt.Sprintf("(%s)", combined))
			} else if len(orParts) == 1 {
				parts = append(parts, orParts[0])
			}

		case "NOT":
			// NOT will be handled separately in the main function
			return "", nil, 0, fmt.Errorf("NOT in nested condition - handle separately")

		default:
			// Field with operators
			opMap, ok := opMapRaw.(map[string]any)
			if !ok {
				return "", nil, 0, fmt.Errorf("field %s value must be a map", field)
			}

			if err := ValidatePathV2(field); err != nil {
				return "", nil, 0, err
			}

			fullPath := pathPrefix + field

			if isOperatorMap(opMap) {
				// Check for array operators - these need separate handling
				hasArrayOp := false
				for op := range opMap {
					if op == "any" || op == "all" {
						hasArrayOp = true
						break
					}
				}
				if hasArrayOp {
					return "", nil, 0, fmt.Errorf("array operators in nested condition - handle separately")
				}

				// Simple operators - create conditions
				// Sort operators for deterministic variable assignment
				opKeys := make([]string, 0, len(opMap))
				for op := range opMap {
					opKeys = append(opKeys, op)
				}
				slices.Sort(opKeys)

				for _, op := range opKeys {
					value := opMap[op]
					cond, err := NewJSONPathCondition(fullPath, op, value)
					if err != nil {
						return "", nil, 0, err
					}
					cond.SetVarName(fmt.Sprintf("v%d", currentVarOffset))
					condStr, val, err := cond.ToJSONPathString()
					if err != nil {
						return "", nil, 0, err
					}
					parts = append(parts, condStr)
					if val != nil {
						vars[cond.varName] = val
					}
					currentVarOffset++
				}
			} else {
				// Nested object - recurse
				condStr, subVars, newOffset, err := buildNestedConditionString(opMap, fullPath+".", currentVarOffset, LogicAnd)
				if err != nil {
					return "", nil, 0, err
				}
				parts = append(parts, condStr)
				for k, v := range subVars {
					vars[k] = v
				}
				currentVarOffset = newOffset
			}
		}
	}

	if len(parts) == 0 {
		return "", vars, currentVarOffset, nil
	}

	if len(parts) == 1 {
		return parts[0], vars, currentVarOffset, nil
	}

	// Combine parts with appropriate connector
	connector := " && "
	if logic == LogicOr {
		connector = " || "
	}

	result := ""
	for i, part := range parts {
		if i > 0 {
			result += connector
		}
		result += part
	}

	return result, vars, currentVarOffset, nil
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > len(substr) && (s[:len(substr)] == substr || 
		s[len(s)-len(substr):] == substr || 
		indexOf(s, substr) >= 0)))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// convertFilterMapWithPrefix is the recursive implementation
// pathPrefix is used for nested objects (e.g., "details." for nested field access)
func convertFilterMapWithPrefix(
	col exp.IdentifierExpression,
	filterMap map[string]any,
	pathPrefix string,
	dialect Dialect,
) (exp.Expression, error) {
	if len(filterMap) == 0 {
		return nil, fmt.Errorf("empty filter map")
	}

	andExprs := make([]exp.Expression, 0)

	// Optimization: Collect all simple field conditions to combine into ONE jsonb_path_exists
	combinedFilter := NewJSONPathFilter(col, dialect)
	hasSimpleConditions := false

	// Sort keys for deterministic output
	keys := make([]string, 0, len(filterMap))
	for k := range filterMap {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	for _, field := range keys {
		opMapRaw := filterMap[field]

		switch field {
		case "AND":
			// Handle AND: array of filter maps
			andFilters, ok := opMapRaw.([]any)
			if !ok {
				return nil, fmt.Errorf("AND must be an array")
			}

			// Try to build a single nested condition string
			// Process each AND element and combine
			andParts := make([]string, 0)
			allVars := make(map[string]any)
			varOffset := 0
			canBuildNested := true

			for _, af := range andFilters {
				afMap, ok := af.(map[string]any)
				if !ok {
					canBuildNested = false
					break
				}

				partStr, partVars, newOffset, err := buildNestedConditionString(afMap, pathPrefix, varOffset, LogicAnd)
				if err != nil {
					canBuildNested = false
					break
				}
				andParts = append(andParts, partStr)
				for k, v := range partVars {
					allVars[k] = v
				}
				varOffset = newOffset
			}

			if canBuildNested && len(andParts) > 0 {
				// Combine AND parts
				combinedStr := ""
				for i, part := range andParts {
					if i > 0 {
						combinedStr += " && "
					}
					// Add parentheses if part contains OR and doesn't already have them
					if contains(part, " || ") && (len(part) == 0 || part[0] != '(' || part[len(part)-1] != ')') {
						combinedStr += fmt.Sprintf("(%s)", part)
					} else {
						combinedStr += part
					}
				}

				// Create a custom JSONPathFilterExpr with the combined string
				jsonPath := fmt.Sprintf("$ ? (%s)", combinedStr)
				andExprs = append(andExprs, dialect.JSONPathExists(col, jsonPath, allVars))
				continue
			}

			// Fallback: process as separate expressions
			complexExprs := make([]exp.Expression, 0)
			for _, af := range andFilters {
				afMap := af.(map[string]any)
				subExpr, err := convertFilterMapWithPrefix(col, afMap, pathPrefix, dialect)
				if err != nil {
					return nil, err
				}
				complexExprs = append(complexExprs, subExpr)
			}

			if len(complexExprs) > 0 {
				combined, err := BuildLogicalFilter(col, LogicAnd, complexExprs, false)
				if err != nil {
					return nil, err
				}
				andExprs = append(andExprs, combined)
			}

		case "OR":
			// Handle OR: array of filter maps
			orFilters, ok := opMapRaw.([]any)
			if !ok {
				return nil, fmt.Errorf("OR must be an array")
			}

			// Try to build a single nested condition string
			orParts := make([]string, 0)
			allVars := make(map[string]any)
			varOffset := 0
			canBuildNested := true

			for _, of := range orFilters {
				ofMap, ok := of.(map[string]any)
				if !ok {
					canBuildNested = false
					break
				}

				partStr, partVars, newOffset, err := buildNestedConditionString(ofMap, pathPrefix, varOffset, LogicOr)
				if err != nil {
					canBuildNested = false
					break
				}
				orParts = append(orParts, partStr)
				for k, v := range partVars {
					allVars[k] = v
				}
				varOffset = newOffset
			}

			if canBuildNested && len(orParts) > 0 {
				// Combine OR parts
				combinedStr := ""
				for i, part := range orParts {
					if i > 0 {
						combinedStr += " || "
					}
					combinedStr += part
				}

				// Create a custom JSONPathFilterExpr with the combined string
				// Add double parentheses for OR (as expected by tests for logical OR operator compatibility)
				jsonPath := fmt.Sprintf("$ ? ((%s))", combinedStr)
				andExprs = append(andExprs, dialect.JSONPathExists(col, jsonPath, allVars))
				continue
			}

			// Fallback: process as separate expressions
			// Fallback: process as separate expressions
			complexExprs := make([]exp.Expression, 0)
			for _, of := range orFilters {
				ofMap := of.(map[string]any)
				subExpr, err := convertFilterMapWithPrefix(col, ofMap, pathPrefix, dialect)
				if err != nil {
					return nil, err
				}
				complexExprs = append(complexExprs, subExpr)
			}

			if len(complexExprs) > 0 {
				combined, err := BuildLogicalFilter(col, LogicOr, complexExprs, false)
				if err != nil {
					return nil, err
				}
				andExprs = append(andExprs, combined)
			}

		case "NOT":
			// Handle NOT: single filter map
			notMap, ok := opMapRaw.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("NOT must be a map")
			}

			// Check if NOT contains simple conditions that can be negated in JSONPath
			simpleConditions, hasComplexity := extractSimpleConditions(notMap, pathPrefix)
			if !hasComplexity && len(simpleConditions) > 0 {
				// Simple case: negate in JSONPath
				notFilter := NewJSONPathFilter(col, dialect)
				notFilter.SetNegate(true)
				for _, cond := range simpleConditions {
					notFilter.AddCondition(cond)
				}
				notExpr, err := notFilter.Expression()
				if err != nil {
					return nil, err
				}
				andExprs = append(andExprs, notExpr)
			} else {
				// Complex case: contains AND/OR or other complex operators
				// For now, fall back to recursive processing and apply De Morgan's laws if needed
				// TODO: Implement De Morgan's laws for nested NOT cases
				subExpr, err := convertFilterMapWithPrefix(col, notMap, pathPrefix, dialect)
				if err != nil {
					return nil, err
				}

				// Check if subExpr is a JSONPathFilterExpr that we can negate
				// For now, wrap in SQL NOT for complex cases
				negated, err := BuildLogicalFilter(col, LogicAnd, []exp.Expression{subExpr}, true)
				if err != nil {
					return nil, err
				}
				andExprs = append(andExprs, negated)
			}

		default:
			// Field with either operators or nested object/array filter
			opMap, ok := opMapRaw.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("field %s value must be a map", field)
			}

			// Validate the field path
			if err := ValidatePathV2(field); err != nil {
				return nil, err
			}

			fullPath := pathPrefix + field

			// Check if this is an operator map or a nested filter
			if isOperatorMap(opMap) {
				// Check if these are simple operators that can be combined
				simpleOps, complexExprs, err := categorizeFieldOperators(col, fullPath, opMap, dialect)
				if err != nil {
					return nil, err
				}

				// Add simple conditions to combined filter
				for _, cond := range simpleOps {
					combinedFilter.AddCondition(cond)
					hasSimpleConditions = true
				}

				// Complex operators (any, all) stay separate
				andExprs = append(andExprs, complexExprs...)
			} else {
				// Nested object filter - recurse with updated path prefix
				subExpr, err := convertFilterMapWithPrefix(col, opMap, fullPath+".", dialect)
				if err != nil {
					return nil, err
				}
				andExprs = append(andExprs, subExpr)
			}
		}
	}

	// Build the combined filter if we have simple conditions
	if hasSimpleConditions {
		combinedExpr, err := combinedFilter.Expression()
		if err != nil {
			return nil, err
		}
		// Prepend combined expression so it comes first
		andExprs = append([]exp.Expression{combinedExpr}, andExprs...)
	}

	if len(andExprs) == 0 {
		return nil, fmt.Errorf("no valid conditions found")
	}

	// Combine all expressions with AND
	if len(andExprs) == 1 {
		return andExprs[0], nil
	}

	return BuildLogicalFilter(col, LogicAnd, andExprs, false)
}

// extractSimpleConditions attempts to extract simple field conditions from a filter map
// Returns (conditions, hasComplexity) where hasComplexity indicates presence of arrays/nested objects/logical ops
func extractSimpleConditions(filterMap map[string]any, pathPrefix string) ([]*JSONPathConditionExpr, bool) {
	conditions := make([]*JSONPathConditionExpr, 0)

	for field, opMapRaw := range filterMap {
		// Check for logical operators - these are complex
		if field == "AND" || field == "OR" || field == "NOT" {
			return nil, true
		}

		opMap, ok := opMapRaw.(map[string]any)
		if !ok {
			return nil, true
		}

		// Validate path
		if err := ValidatePathV2(field); err != nil {
			return nil, true
		}

		fullPath := pathPrefix + field

		// Check if this is an operator map
		if !isOperatorMap(opMap) {
			// Nested object - complex
			return nil, true
		}

		// Check for array operators - these are complex
		for op := range opMap {
			if op == "any" || op == "all" {
				return nil, true
			}
		}

		// All operators are simple - extract conditions
		for op, value := range opMap {
			cond, err := NewJSONPathCondition(fullPath, op, value)
			if err != nil {
				return nil, true
			}
			conditions = append(conditions, cond)
		}
	}

	return conditions, false
}

// categorizeFieldOperators separates simple operators (can be combined) from complex ones (need separate handling)
// Returns: (simple conditions, complex expressions, error)
func categorizeFieldOperators(
	col exp.IdentifierExpression,
	fieldPath string,
	opMap map[string]any,
	dialect Dialect,
) ([]*JSONPathConditionExpr, []exp.Expression, error) {
	simpleConditions := make([]*JSONPathConditionExpr, 0)
	complexExprs := make([]exp.Expression, 0)

	// Sort operators for deterministic output
	opKeys := make([]string, 0, len(opMap))
	for op := range opMap {
		opKeys = append(opKeys, op)
	}
	slices.Sort(opKeys)

	for _, op := range opKeys {
		value := opMap[op]

		switch op {
		case "any":
			// Array filter: any element matches - needs separate handling
			anyFilter, ok := value.(map[string]any)
			if !ok {
				return nil, nil, fmt.Errorf("'any' operator value must be a map")
			}

			arrayFilter, err := NewJSONArrayFilter(col, fieldPath, ArrayAny, dialect)
			if err != nil {
				return nil, nil, fmt.Errorf("creating array filter: %w", err)
			}

			conditions, err := convertFilterToConditions(anyFilter, "")
			if err != nil {
				return nil, nil, fmt.Errorf("processing 'any' filter: %w", err)
			}

			for _, cond := range conditions {
				arrayFilter.AddCondition(cond)
			}

			expr, err := arrayFilter.Expression()
			if err != nil {
				return nil, nil, err
			}
			complexExprs = append(complexExprs, expr)

		case "all":
			// Array filter: all elements match - needs separate handling
			allFilter, ok := value.(map[string]any)
			if !ok {
				return nil, nil, fmt.Errorf("'all' operator value must be a map")
			}

			arrayFilter, err := NewJSONArrayFilter(col, fieldPath, ArrayAll, dialect)
			if err != nil {
				return nil, nil, fmt.Errorf("creating array filter: %w", err)
			}

			conditions, err := convertFilterToConditions(allFilter, "")
			if err != nil {
				return nil, nil, fmt.Errorf("processing 'all' filter: %w", err)
			}

			for _, cond := range conditions {
				arrayFilter.AddCondition(cond)
			}

			expr, err := arrayFilter.Expression()
			if err != nil {
				return nil, nil, err
			}
			complexExprs = append(complexExprs, expr)

		default:
			// Simple operators (eq, neq, gt, gte, lt, lte, like, isNull) - can be combined
			cond, err := NewJSONPathCondition(fieldPath, op, value)
			if err != nil {
				return nil, nil, fmt.Errorf("field %s: %w", fieldPath, err)
			}
			simpleConditions = append(simpleConditions, cond)
		}
	}

	return simpleConditions, complexExprs, nil
}

// convertFilterToConditions converts a filter map to a list of conditions
func convertFilterToConditions(filterMap map[string]any, pathPrefix string) ([]*JSONPathConditionExpr, error) {
	conditions := make([]*JSONPathConditionExpr, 0)

	// Sort keys for deterministic output
	keys := make([]string, 0, len(filterMap))
	for k := range filterMap {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	for _, field := range keys {
		opMapRaw := filterMap[field]

		// Skip logical operators for now (they need special handling)
		if field == "AND" || field == "OR" || field == "NOT" {
			continue
		}

		opMap, ok := opMapRaw.(map[string]any)
		if !ok {
			continue
		}

		if err := ValidatePathV2(field); err != nil {
			return nil, err
		}

		fullPath := pathPrefix + field

		if isOperatorMap(opMap) {
			// Process operators
			for op, value := range opMap {
				cond, err := NewJSONPathCondition(fullPath, op, value)
				if err != nil {
					return nil, err
				}
				conditions = append(conditions, cond)
			}
		} else {
			// Nested object - recurse
			nestedConds, err := convertFilterToConditions(opMap, fullPath+".")
			if err != nil {
				return nil, err
			}
			conditions = append(conditions, nestedConds...)
		}
	}

	return conditions, nil
}

// ConvertMapComparatorToExpression converts a MapComparator filter to an expression
// This replaces ParseMapComparator + BuildMapFilter
func ConvertMapComparatorToExpression(
	col exp.IdentifierExpression,
	filterMap map[string]any,
	dialect Dialect,
) (exp.Expression, error) {
	exprs := make([]exp.Expression, 0)

	// Handle isNull
	if isNullRaw, ok := filterMap["isNull"]; ok {
		isNull := cast.ToBool(isNullRaw)
		nullCheck := NewJSONNullCheck(col, isNull, dialect)
		expr, err := nullCheck.Expression()
		if err != nil {
			return nil, err
		}
		exprs = append(exprs, expr)
	}

	// Handle contains (@>)
	if containsRaw, ok := filterMap["contains"]; ok {
		contains, ok := containsRaw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("contains must be a map")
		}
		containsExpr := NewJSONContains(col, contains, dialect)
		expr, err := containsExpr.Expression()
		if err != nil {
			return nil, err
		}
		exprs = append(exprs, expr)
	}

	// Handle where (AND conditions)
	if whereRaw, ok := filterMap["where"]; ok {
		where, ok := whereRaw.([]any)
		if !ok {
			return nil, fmt.Errorf("where must be an array")
		}

		conditions, err := parsePathConditionsV2(where)
		if err != nil {
			return nil, fmt.Errorf("parsing where: %w", err)
		}

		if len(conditions) > 0 {
			filter := NewJSONPathFilter(col, dialect)
			filter.SetLogic(LogicAnd)
			for _, cond := range conditions {
				filter.AddCondition(cond)
			}
			expr, err := filter.Expression()
			if err != nil {
				return nil, err
			}
			exprs = append(exprs, expr)
		}
	}

	// Handle whereAny (OR conditions)
	if whereAnyRaw, ok := filterMap["whereAny"]; ok {
		whereAny, ok := whereAnyRaw.([]any)
		if !ok {
			return nil, fmt.Errorf("whereAny must be an array")
		}

		conditions, err := parsePathConditionsV2(whereAny)
		if err != nil {
			return nil, fmt.Errorf("parsing whereAny: %w", err)
		}

		if len(conditions) > 0 {
			filter := NewJSONPathFilter(col, dialect)
			filter.SetLogic(LogicOr)
			for _, cond := range conditions {
				filter.AddCondition(cond)
			}
			expr, err := filter.Expression()
			if err != nil {
				return nil, err
			}
			exprs = append(exprs, expr)
		}
	}

	if len(exprs) == 0 {
		return nil, fmt.Errorf("no valid conditions in MapComparator")
	}

	// Combine all with AND
	if len(exprs) == 1 {
		return exprs[0], nil
	}

	return BuildLogicalFilter(col, LogicAnd, exprs, false)
}

// parsePathConditionsV2 parses an array of condition maps into JSONPathConditionExpr slice
func parsePathConditionsV2(conditions []any) ([]*JSONPathConditionExpr, error) {
	result := make([]*JSONPathConditionExpr, 0, len(conditions))

	for _, c := range conditions {
		condMap, ok := c.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("condition must be a map")
		}

		path, ok := condMap["path"].(string)
		if !ok {
			return nil, fmt.Errorf("condition must have a 'path' string field")
		}

		// Find the operator and value
		for op, value := range condMap {
			if op == "path" {
				continue
			}

			cond, err := NewJSONPathCondition(path, op, value)
			if err != nil {
				return nil, err
			}
			result = append(result, cond)
		}
	}

	return result, nil
}
