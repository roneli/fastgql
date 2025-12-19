package sql

import (
	"fmt"
	"slices"
	"strings"

	"github.com/doug-martin/goqu/v9/exp"
)

// ConvertFilterMapToExpression converts a FilterInput-style map to a JSON filter expression
// This replaces BuildJsonFilterFromOperatorMap with expression-based building
func ConvertFilterMapToExpression(
	col exp.IdentifierExpression,
	filterMap map[string]any,
	dialect Dialect,
) (exp.Expression, error) {
	return buildJSONFilterExpression(col, filterMap, "", dialect)
}

// buildJSONPathString recursively builds a JSONPath condition string from a filter map
// This is used for nested AND/OR cases that need to be combined into a single JSONPath expression
// Returns the condition string (e.g., "@.field == $v0 && (@.field2 > $v1 || @.field2 < $v2)"),
// variables map for parameterization, and the next variable offset
func buildJSONPathString(
	filterMap map[string]any,
	pathPrefix string,
	varOffset int,
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

				condStr, subVars, newOffset, err := buildJSONPathString(afMap, pathPrefix, currentVarOffset)
				if err != nil {
					return "", nil, 0, err
				}
				andParts = append(andParts, condStr)
				for k, v := range subVars {
					vars[k] = v
				}
				currentVarOffset = newOffset
			}

			if len(andParts) > 0 {
				combined := ""
				for i, part := range andParts {
					if i > 0 {
						combined += " && "
					}
					// Add parentheses if part contains OR and doesn't already have them
					if strings.Contains(part, " || ") && (len(part) == 0 || part[0] != '(' || part[len(part)-1] != ')') {
						combined += fmt.Sprintf("(%s)", part)
					} else {
						combined += part
					}
				}
				parts = append(parts, combined)
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

				condStr, subVars, newOffset, err := buildJSONPathString(ofMap, pathPrefix, currentVarOffset)
				if err != nil {
					return "", nil, 0, err
				}
				orParts = append(orParts, condStr)
				for k, v := range subVars {
					vars[k] = v
				}
				currentVarOffset = newOffset
			}

			if len(orParts) > 0 {
				combined := ""
				for i, part := range orParts {
					if i > 0 {
						combined += " || "
					}
					combined += part
				}
				parts = append(parts, fmt.Sprintf("(%s)", combined))
			}

		case "NOT":
			return "", nil, 0, fmt.Errorf("NOT in nested JSONPath string - handle separately")

		default:
			// Field with operators
			opMap, ok := opMapRaw.(map[string]any)
			if !ok {
				return "", nil, 0, fmt.Errorf("field %s value must be a map", field)
			}

			if err := validatePath(field); err != nil {
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
					return "", nil, 0, fmt.Errorf("array operators in nested JSONPath - handle separately")
				}

				// Simple operators - create conditions
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
				condStr, subVars, newOffset, err := buildJSONPathString(opMap, fullPath+".", currentVarOffset)
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

	// Combine parts with AND (default for top-level)
	result := ""
	for i, part := range parts {
		if i > 0 {
			result += " && "
		}
		result += part
	}

	return result, vars, currentVarOffset, nil
}

// buildJSONFilterExpression recursively builds a JSON filter expression from a filter map
// pathPrefix tracks the current JSON path depth for nested objects (e.g., "details." for nested field access)
// This function follows the same recursive pattern as buildFilterExp in builder.go
func buildJSONFilterExpression(
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

			// Try to build a single JSONPath string for nested AND/OR cases
			// This handles cases like AND: [{field: {eq: "x"}}, {OR: [{field2: {gt: 1}}, {field2: {lt: 2}}]}]
			condStr, allVars, _, err := buildJSONPathString(
				map[string]any{"AND": andFilters},
				pathPrefix,
				0,
			)
			if err == nil && condStr != "" {
				// Successfully built JSONPath string - use it
				jsonPath := fmt.Sprintf("$ ? (%s)", condStr)
				andExprs = append(andExprs, dialect.JSONPathExists(col, jsonPath, allVars))
			} else {
				// Fallback: try simple condition extraction
				allConditions := make([]*JSONPathConditionExpr, 0)
				canCombine := true
				for _, af := range andFilters {
					afMap, ok := af.(map[string]any)
					if !ok {
						canCombine = false
						break
					}
					conditions, hasComplexity := extractSimpleConditions(afMap, pathPrefix)
					if hasComplexity || len(conditions) == 0 {
						canCombine = false
						break
					}
					allConditions = append(allConditions, conditions...)
				}

				if canCombine && len(allConditions) > 0 {
					// Combine all conditions into a single JSONPath filter
					andFilter := NewJSONPathFilter(col, dialect)
					for _, cond := range allConditions {
						andFilter.AddCondition(cond)
					}
					expr, err := andFilter.Expression()
					if err != nil {
						return nil, err
					}
					andExprs = append(andExprs, expr)
				} else {
					// Fallback: process recursively and combine with SQL AND
					complexExprs := make([]exp.Expression, 0)
					for _, af := range andFilters {
						afMap, ok := af.(map[string]any)
						if !ok {
							return nil, fmt.Errorf("AND element must be a map")
						}
						subExpr, err := buildJSONFilterExpression(col, afMap, pathPrefix, dialect)
						if err != nil {
							return nil, err
						}
						complexExprs = append(complexExprs, subExpr)
					}

					if len(complexExprs) > 0 {
						combined, err := BuildLogicalFilter(col, exp.AndType, complexExprs, false)
						if err != nil {
							return nil, err
						}
						andExprs = append(andExprs, combined)
					}
				}
			}

		case "OR":
			// Handle OR: array of filter maps
			orFilters, ok := opMapRaw.([]any)
			if !ok {
				return nil, fmt.Errorf("OR must be an array")
			}

			// Try to extract simple conditions from each OR element
			orConditionGroups := make([][]*JSONPathConditionExpr, 0)
			canCombine := true
			for _, of := range orFilters {
				ofMap, ok := of.(map[string]any)
				if !ok {
					canCombine = false
					break
				}
				conditions, hasComplexity := extractSimpleConditions(ofMap, pathPrefix)
				if hasComplexity || len(conditions) == 0 {
					canCombine = false
					break
				}
				orConditionGroups = append(orConditionGroups, conditions)
			}

			if canCombine && len(orConditionGroups) > 0 {
				// Combine all OR condition groups into a single JSONPath filter
				// Build JSONPath string manually since JSONPathFilterExpr only supports AND
				vars := make(map[string]any)
				orParts := make([]string, 0)

				varIndex := 0
				for _, conditionGroup := range orConditionGroups {
					// Each OR element becomes a condition group (may have multiple conditions with AND)
					groupParts := make([]string, 0)
					for _, cond := range conditionGroup {
						// Check if this operator needs a variable
						needsVariable := cond.operator != "prefix" && cond.operator != "suffix" &&
							cond.operator != "ilike" && cond.operator != "contains" && cond.operator != "isNull"

						if needsVariable {
							varName := fmt.Sprintf("v%d", varIndex)
							cond.SetVarName(varName)
							varIndex++
						}

						condStr, val, err := cond.ToJSONPathString()
						if err != nil {
							return nil, err
						}
						groupParts = append(groupParts, condStr)
						if val != nil {
							vars[cond.varName] = val
						}
					}

					// Combine conditions in this group with AND
					if len(groupParts) == 1 {
						orParts = append(orParts, groupParts[0])
					} else {
						combinedGroup := ""
						for i, part := range groupParts {
							if i > 0 {
								combinedGroup += " && "
							}
							combinedGroup += part
						}
						orParts = append(orParts, combinedGroup)
					}
				}

				// Combine OR parts
				combinedConditions := ""
				for i, part := range orParts {
					if i > 0 {
						combinedConditions += " || "
					}
					combinedConditions += part
				}

				// Wrap in double parentheses as expected by tests
				jsonPath := fmt.Sprintf("$ ? ((%s))", combinedConditions)
				andExprs = append(andExprs, dialect.JSONPathExists(col, jsonPath, vars))
			} else {
				// Fallback: process recursively and combine with SQL OR
				complexExprs := make([]exp.Expression, 0)
				for _, of := range orFilters {
					ofMap, ok := of.(map[string]any)
					if !ok {
						return nil, fmt.Errorf("OR element must be a map")
					}
					subExpr, err := buildJSONFilterExpression(col, ofMap, pathPrefix, dialect)
					if err != nil {
						return nil, err
					}
					complexExprs = append(complexExprs, subExpr)
				}

				if len(complexExprs) > 0 {
					combined, err := BuildLogicalFilter(col, exp.OrType, complexExprs, false)
					if err != nil {
						return nil, err
					}
					andExprs = append(andExprs, combined)
				}
			}

		case "NOT":
			// Handle NOT: single filter map
			notMap, ok := opMapRaw.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("NOT must be a map")
			}

			// Check if NOT contains only simple conditions that can be negated in JSONPath
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
				// Complex case: process recursively and wrap with SQL NOT
				subExpr, err := buildJSONFilterExpression(col, notMap, pathPrefix, dialect)
				if err != nil {
					return nil, err
				}

				negated, err := BuildLogicalFilter(col, exp.AndType, []exp.Expression{subExpr}, true)
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
			if err := validatePath(field); err != nil {
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
				subExpr, err := buildJSONFilterExpression(col, opMap, fullPath+".", dialect)
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

	return BuildLogicalFilter(col, exp.AndType, andExprs, false)
}

// extractSimpleConditions attempts to extract simple field conditions from a filter map
// Simple conditions are field operators (eq, neq, gt, etc.) that can be combined into a single JSONPath filter
// Returns (conditions, hasComplexity) where hasComplexity indicates presence of:
// - Logical operators (AND/OR/NOT)
// - Array operators (any/all)
// - Nested objects
// If hasComplexity is true, the conditions cannot be simply combined and need recursive processing
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
		if err := validatePath(field); err != nil {
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

			arrayFilter, err := NewJSONArrayFilter(col, fieldPath, arrayAny, dialect)
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

			arrayFilter, err := NewJSONArrayFilter(col, fieldPath, arrayAll, dialect)
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

		if err := validatePath(field); err != nil {
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
