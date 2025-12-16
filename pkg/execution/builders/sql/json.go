package sql

import (
	"fmt"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/roneli/fastgql/pkg/execution/builders"
)

// jsonPathOpMap maps GraphQL operators to JSONPath operators
var jsonPathOpMap = map[string]string{
	"eq":   "==",
	"neq":  "!=",
	"gt":   ">",
	"gte":  ">=",
	"lt":   "<",
	"lte":  "<=",
	"like": "like_regex",
}

// knownOperators is used to detect if a map contains operators or nested fields
var knownOperators = map[string]bool{
	"eq": true, "neq": true, "gt": true, "gte": true,
	"lt": true, "lte": true, "like": true, "ilike": true,
	"isNull": true, "in": true, "notIn": true,
	"contains": true, "prefix": true, "suffix": true,
	// Array operators
	"any": true, "all": true,
}

// buildJsonFieldObject builds an expression to extract selected JSON fields
// Uses jsonb_path_query_first for efficient extraction, jsonb_build_object for construction
func buildJsonFieldObject(
	baseCol exp.Expression,
	selections builders.Fields,
	pathPrefix string,
	dialect string,
) (exp.Expression, error) {
	if len(selections) == 0 {
		// No selections - return the entire JSON column
		return baseCol, nil
	}

	// For multiple fields or mixed types, use jsonb_build_object with jsonb_path_query_first
	args := make([]interface{}, 0, len(selections)*2)
	for _, sel := range selections {
		args = append(args, goqu.L(fmt.Sprintf("'%s'", sel.Name)))

		var valueExpr exp.Expression

		switch sel.FieldType {
		case builders.TypeScalar:
			// Extract scalar: use native -> operator for efficiency (faster than jsonb_path_query_first)
			// For simple paths, -> is more efficient as it's a native operator
			if err := ValidatePathV2(sel.Name); err != nil {
				return nil, fmt.Errorf("invalid JSON field name %s: %w", sel.Name, err)
			}
			// Build path using -> operator: col->'field' for JSONB, or col->>'field' for text
			// We use -> to get JSONB, which works well with jsonb_build_object
			valueExpr = goqu.L("?->?", baseCol, sel.Name)

		case builders.TypeObject, builders.TypeJson:
			// Nested object: extract the nested JSON object first, then recursively build
			if err := ValidatePathV2(sel.Name); err != nil {
				return nil, fmt.Errorf("invalid JSON field name %s: %w", sel.Name, err)
			}
			// Extract the nested object using -> operator (more efficient than jsonb_path_query_first for simple paths)
			nestedCol := goqu.L("?->?", baseCol, sel.Name)
			// Recursively build the nested object structure
			nestedObj, err := buildJsonFieldObject(nestedCol, sel.Selections, "", dialect)
			if err != nil {
				return nil, fmt.Errorf("building nested JSON for %s: %w", sel.Name, err)
			}
			valueExpr = nestedObj

		default:
			return nil, fmt.Errorf("unsupported field type %s in JSON selection", sel.FieldType)
		}

		args = append(args, valueExpr)
	}

	sqlDialect := GetSQLDialect(dialect)
	return sqlDialect.JSONBuildObject(args...), nil
}

// isOperatorMap checks if a map contains operators (eq, neq, etc.) vs nested field filters
func isOperatorMap(m map[string]any) bool {
	for k := range m {
		if knownOperators[k] {
			return true
		}
	}
	return false
}

// toJsonPathOp converts a GraphQL operator to JSONPath operator
func toJsonPathOp(op string) (string, error) {
	if jpOp, ok := jsonPathOpMap[op]; ok {
		return jpOp, nil
	}
	return "", fmt.Errorf("unsupported operator: %s", op)
}
