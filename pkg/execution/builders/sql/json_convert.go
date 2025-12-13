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

			subExprs := make([]exp.Expression, 0, len(andFilters))
			for _, af := range andFilters {
				afMap, ok := af.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("AND element must be a map")
				}
				subExpr, err := convertFilterMapWithPrefix(col, afMap, pathPrefix, dialect)
				if err != nil {
					return nil, err
				}
				subExprs = append(subExprs, subExpr)
			}

			if len(subExprs) > 0 {
				combined, err := BuildLogicalFilter(col, LogicAnd, subExprs, false)
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

			subExprs := make([]exp.Expression, 0, len(orFilters))
			for _, of := range orFilters {
				ofMap, ok := of.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("OR element must be a map")
				}
				subExpr, err := convertFilterMapWithPrefix(col, ofMap, pathPrefix, dialect)
				if err != nil {
					return nil, err
				}
				subExprs = append(subExprs, subExpr)
			}

			if len(subExprs) > 0 {
				combined, err := BuildLogicalFilter(col, LogicOr, subExprs, false)
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

			subExpr, err := convertFilterMapWithPrefix(col, notMap, pathPrefix, dialect)
			if err != nil {
				return nil, err
			}

			// Wrap in NOT
			negated, err := BuildLogicalFilter(col, LogicAnd, []exp.Expression{subExpr}, true)
			if err != nil {
				return nil, err
			}
			andExprs = append(andExprs, negated)

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
				// Process operators for this field
				fieldExprs, err := processFieldOperatorsV2(col, fullPath, opMap, dialect)
				if err != nil {
					return nil, err
				}
				andExprs = append(andExprs, fieldExprs...)
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

	if len(andExprs) == 0 {
		return nil, fmt.Errorf("no valid conditions found")
	}

	// Combine all expressions with AND
	if len(andExprs) == 1 {
		return andExprs[0], nil
	}

	return BuildLogicalFilter(col, LogicAnd, andExprs, false)
}

// processFieldOperatorsV2 processes operators for a single field using new expression types
func processFieldOperatorsV2(
	col exp.IdentifierExpression,
	fieldPath string,
	opMap map[string]any,
	dialect Dialect,
) ([]exp.Expression, error) {
	exprs := make([]exp.Expression, 0)

	// Sort operators for deterministic output
	opKeys := make([]string, 0, len(opMap))
	for op := range opMap {
		opKeys = append(opKeys, op)
	}
	slices.Sort(opKeys)

	for _, op := range opKeys {
		value := opMap[op]

		switch op {
		case "isNull":
			// Handle NULL check
			isNull := cast.ToBool(value)
			cond, err := NewJSONPathCondition(fieldPath, "isNull", isNull)
			if err != nil {
				return nil, err
			}

			filter := NewJSONPathFilter(col, dialect)
			filter.AddCondition(cond)
			expr, err := filter.Expression()
			if err != nil {
				return nil, err
			}
			exprs = append(exprs, expr)

		case "any":
			// Array filter: any element matches the condition
			anyFilter, ok := value.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("'any' operator value must be a map")
			}

			// Build array filter
			arrayFilter, err := NewJSONArrayFilter(col, fieldPath, ArrayAny, dialect)
			if err != nil {
				return nil, fmt.Errorf("creating array filter: %w", err)
			}

			// Convert the nested filter to conditions
			conditions, err := convertFilterToConditions(anyFilter, "")
			if err != nil {
				return nil, fmt.Errorf("processing 'any' filter: %w", err)
			}

			for _, cond := range conditions {
				arrayFilter.AddCondition(cond)
			}

			expr, err := arrayFilter.Expression()
			if err != nil {
				return nil, err
			}
			exprs = append(exprs, expr)

		case "all":
			// Array filter: all elements match the condition
			// FIX FOR BUG: Implement proper 'all' logic
			allFilter, ok := value.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("'all' operator value must be a map")
			}

			// Build array filter with ALL mode
			arrayFilter, err := NewJSONArrayFilter(col, fieldPath, ArrayAll, dialect)
			if err != nil {
				return nil, fmt.Errorf("creating array filter: %w", err)
			}

			// Convert the nested filter to conditions
			conditions, err := convertFilterToConditions(allFilter, "")
			if err != nil {
				return nil, fmt.Errorf("processing 'all' filter: %w", err)
			}

			for _, cond := range conditions {
				arrayFilter.AddCondition(cond)
			}

			expr, err := arrayFilter.Expression()
			if err != nil {
				return nil, err
			}
			exprs = append(exprs, expr)

		default:
			// Standard operator (eq, neq, gt, gte, lt, lte, like)
			cond, err := NewJSONPathCondition(fieldPath, op, value)
			if err != nil {
				return nil, fmt.Errorf("field %s: %w", fieldPath, err)
			}

			filter := NewJSONPathFilter(col, dialect)
			filter.AddCondition(cond)
			expr, err := filter.Expression()
			if err != nil {
				return nil, err
			}
			exprs = append(exprs, expr)
		}
	}

	return exprs, nil
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
