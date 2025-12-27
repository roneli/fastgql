package sql

import (
	"fmt"
	"slices"

	"github.com/doug-martin/goqu/v9/exp"
)

// ConvertFilterMapToExpression converts a FilterInput-style map to a goqu Expression
// by building JSONPath expressions and combining them.
//
// Example:
//
//	Input:  {color: {eq: "red"}, size: {gt: 5}, tags: {any: {eq: "featured"}}}
//	Output: jsonb_path_exists(col, '$ ? (@.color == $v0 && @.size > $v1 && @.tags[*] == $v2)', vars)
func ConvertFilterMapToExpression(col exp.IdentifierExpression, filterMap map[string]any, dialect Dialect) (exp.Expression, error) {
	if len(filterMap) == 0 {
		return nil, fmt.Errorf("empty filter map")
	}

	exprs, err := buildExprs(filterMap)
	if err != nil {
		return nil, err
	}

	if len(exprs) == 0 {
		return nil, fmt.Errorf("no conditions to build")
	}

	// Wrap all expressions in AND
	root := &LogicalExpr{Op: JSONPathAnd, Children: exprs}

	pathBuilder := NewJSONPathBuilder()
	condStr := pathBuilder.Build(root)

	if condStr == "" {
		return nil, fmt.Errorf("no valid conditions")
	}

	jsonPath := fmt.Sprintf("$ ? (%s)", condStr)
	return dialect.JSONPathExists(col, jsonPath, pathBuilder.Vars()), nil
}

// buildExprs recursively converts a filter map to JSONPathExpr slice
func buildExprs(filterMap map[string]any) ([]JSONPathExpr, error) {
	var exprs []JSONPathExpr
	keys := sortedKeys(filterMap)

	for _, field := range keys {
		value := filterMap[field]

		switch field {
		case "AND":
			expr, err := buildAND(value)
			if err != nil {
				return nil, err
			}
			exprs = append(exprs, expr)

		case "OR":
			expr, err := buildOR(value)
			if err != nil {
				return nil, err
			}
			exprs = append(exprs, expr)

		case "NOT":
			expr, err := buildNOT(value)
			if err != nil {
				return nil, err
			}
			exprs = append(exprs, expr)

		default:
			fieldExprs, err := buildField(field, value)
			if err != nil {
				return nil, err
			}
			exprs = append(exprs, fieldExprs...)
		}
	}

	return exprs, nil
}

func buildAND(value any) (JSONPathExpr, error) {
	filters, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("AND must be an array")
	}

	var children []JSONPathExpr
	for _, f := range filters {
		fMap, ok := f.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("AND element must be a map")
		}
		exprs, err := buildExprs(fMap)
		if err != nil {
			return nil, err
		}
		children = append(children, exprs...)
	}
	return JsonAnd(children...), nil
}

func buildOR(value any) (JSONPathExpr, error) {
	filters, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("OR must be an array")
	}

	var children []JSONPathExpr
	for _, f := range filters {
		fMap, ok := f.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("OR element must be a map")
		}
		exprs, err := buildExprs(fMap)
		if err != nil {
			return nil, err
		}
		// Wrap each OR branch in AND if multiple conditions
		if len(exprs) == 1 {
			children = append(children, exprs[0])
		} else {
			children = append(children, JsonAnd(exprs...))
		}
	}
	return JsonOr(children...), nil
}

func buildNOT(value any) (JSONPathExpr, error) {
	notMap, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("NOT must be a map")
	}

	exprs, err := buildExprs(notMap)
	if err != nil {
		return nil, err
	}

	if len(exprs) == 1 {
		return JsonNot(exprs[0]), nil
	}
	return JsonNot(JsonAnd(exprs...)), nil
}

func buildField(field string, value any) ([]JSONPathExpr, error) {
	opMap, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("field %s value must be a map", field)
	}

	if isOperatorMap(opMap) {
		return buildOperators(field, opMap)
	}

	// Nested object - build path
	return buildNestedField(field, opMap)
}

func buildOperators(path string, opMap map[string]any) ([]JSONPathExpr, error) {
	var exprs []JSONPathExpr

	for _, op := range sortedKeys(opMap) {
		value := opMap[op]

		switch op {
		case "any":
			anyMap, ok := value.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("'any' operator value must be a map")
			}
			expr, err := buildArrayFilter(path, anyMap, false)
			if err != nil {
				return nil, err
			}
			exprs = append(exprs, expr)

		case "all":
			allMap, ok := value.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("'all' operator value must be a map")
			}
			expr, err := buildArrayFilter(path, allMap, true)
			if err != nil {
				return nil, err
			}
			exprs = append(exprs, expr)

		default:
			expr, err := JsonExpr(path, op, value)
			if err != nil {
				return nil, err
			}
			exprs = append(exprs, expr)
		}
	}
	return exprs, nil
}

// buildNestedField recursively builds expressions for nested JSON fields
func buildNestedField(basePath string, filterMap map[string]any) ([]JSONPathExpr, error) {
	var exprs []JSONPathExpr

	for _, field := range sortedKeys(filterMap) {
		value := filterMap[field]
		var fullPath string
		if basePath == "" {
			fullPath = field
		} else {
			fullPath = basePath + "." + field
		}

		opMap, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("field %s value must be a map", field)
		}

		if isOperatorMap(opMap) {
			fieldExprs, err := buildOperators(fullPath, opMap)
			if err != nil {
				return nil, err
			}
			exprs = append(exprs, fieldExprs...)
		} else {
			fieldExprs, err := buildNestedField(fullPath, opMap)
			if err != nil {
				return nil, err
			}
			exprs = append(exprs, fieldExprs...)
		}
	}
	return exprs, nil
}

func buildArrayFilter(arrayPath string, filterMap map[string]any, isAll bool) (JSONPathExpr, error) {
	var innerExprs []JSONPathExpr

	if isOperatorMap(filterMap) {
		// Simple: {eq: "value"}
		for _, op := range sortedKeys(filterMap) {
			expr, err := JsonExpr("", op, filterMap[op])
			if err != nil {
				return nil, err
			}
			innerExprs = append(innerExprs, expr)
		}
	} else {
		// Nested: {field: {eq: "value"}}
		// Use buildNestedField with empty base path since we're inside an array
		nestedExprs, err := buildNestedField("", filterMap)
		if err != nil {
			return nil, err
		}
		innerExprs = append(innerExprs, nestedExprs...)
	}

	if isAll {
		return JsonAll(arrayPath, innerExprs...), nil
	}
	return JsonAny(arrayPath, innerExprs...), nil
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
