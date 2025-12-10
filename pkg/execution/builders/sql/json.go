package sql

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/roneli/fastgql/pkg/execution/builders"
	"github.com/spf13/cast"
)

// JsonPathCondition represents a single condition for Map scalar filtering
type JsonPathCondition struct {
	Path  string // JSON path: "price", "items[0].name", "nested.field"
	Op    string // Operator: eq, neq, gt, gte, lt, lte, like, isNull
	Value any    // The comparison value
}

// JsonFilter represents the full filter for a Map scalar column
type JsonFilter struct {
	Contains map[string]any      // For @> operator: partial JSON match
	Where    []JsonPathCondition // AND conditions
	WhereAny []JsonPathCondition // OR conditions
	IsNull   *bool               // NULL check
}

// pathValidationRegex validates JSON path expressions to prevent injection
// Allows: field, field.nested, field[0], field[0].nested, etc.
var pathValidationRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*(\[[0-9]+\])?(\.[a-zA-Z_][a-zA-Z0-9_]*(\[[0-9]+\])?)*$`)

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

// ValidatePath ensures the path is safe and doesn't contain injection attempts
func ValidatePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}
	if !pathValidationRegex.MatchString(path) {
		return fmt.Errorf("invalid path format: %s", path)
	}
	return nil
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

// BuildJsonPathExpression builds a combined JSONPath expression from conditions
// logic should be "AND" or "OR"
// Returns the JSONPath string and a map of variables for parameterized query
func BuildJsonPathExpression(conditions []JsonPathCondition, logic string) (string, map[string]any, error) {
	if len(conditions) == 0 {
		return "", nil, fmt.Errorf("no conditions provided")
	}

	var parts = make([]string, 0)
	vars := make(map[string]any)

	for i, cond := range conditions {
		if err := ValidatePath(cond.Path); err != nil {
			return "", nil, err
		}

		varName := fmt.Sprintf("v%d", i)

		// Handle isNull specially
		if cond.Op == "isNull" {
			isNull := cast.ToBool(cond.Value)
			if isNull {
				parts = append(parts, fmt.Sprintf("@.%s == null", cond.Path))
			} else {
				parts = append(parts, fmt.Sprintf("@.%s != null", cond.Path))
			}
			continue
		}

		jpOp, err := toJsonPathOp(cond.Op)
		if err != nil {
			return "", nil, err
		}

		parts = append(parts, fmt.Sprintf("@.%s %s $%s", cond.Path, jpOp, varName))
		vars[varName] = cond.Value
	}

	connector := " && "
	if strings.ToUpper(logic) == "OR" {
		connector = " || "
	}

	jsonPath := fmt.Sprintf("$ ? (%s)", strings.Join(parts, connector))
	return jsonPath, vars, nil
}

// BuildJsonFilterFromOperatorMap converts a standard FilterInput-style map to JSONPath
// This is used for @json object fields that use the same filter structure as relations
// Input: {"color": {"eq": "red"}, "size": {"gt": 10}, "AND": [...], "OR": [...]}
// Also supports nested objects: {"details": {"manufacturer": {"eq": "Acme"}}}
// And arrays: {"items": {"any": {"name": {"eq": "widget"}}}}
// Output: JSONPath expression and variables
func BuildJsonFilterFromOperatorMap(filterMap map[string]any) (string, map[string]any, error) {
	return buildJsonFilterFromOperatorMapWithPrefix(filterMap, "")
}

// buildJsonFilterFromOperatorMapWithPrefix is the internal recursive implementation
// pathPrefix is used for nested objects (e.g., "details." for nested field access)
func buildJsonFilterFromOperatorMapWithPrefix(filterMap map[string]any, pathPrefix string) (string, map[string]any, error) {
	if len(filterMap) == 0 {
		return "", nil, fmt.Errorf("empty filter map")
	}

	allConditions := []string{}
	vars := make(map[string]any)
	varIdx := 0

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
				return "", nil, fmt.Errorf("AND must be an array")
			}
			for _, af := range andFilters {
				afMap, ok := af.(map[string]any)
				if !ok {
					return "", nil, fmt.Errorf("AND element must be a map")
				}
				subPath, subVars, err := buildJsonFilterFromOperatorMapWithPrefix(afMap, pathPrefix)
				if err != nil {
					return "", nil, err
				}
				// Extract the condition part from "$ ? (condition)"
				condPart := extractConditionPart(subPath)
				if condPart != "" {
					allConditions = append(allConditions, condPart)
				}
				// Merge vars with offset
				for k, v := range subVars {
					newKey := fmt.Sprintf("v%d", varIdx)
					condPart = strings.Replace(allConditions[len(allConditions)-1], "$"+k, "$"+newKey, 1)
					allConditions[len(allConditions)-1] = condPart
					vars[newKey] = v
					varIdx++
				}
			}

		case "OR":
			// Handle OR: array of filter maps, combine with ||
			orFilters, ok := opMapRaw.([]any)
			if !ok {
				return "", nil, fmt.Errorf("OR must be an array")
			}
			var orParts []string
			for _, of := range orFilters {
				ofMap, ok := of.(map[string]any)
				if !ok {
					return "", nil, fmt.Errorf("OR element must be a map")
				}
				subPath, subVars, err := buildJsonFilterFromOperatorMapWithPrefix(ofMap, pathPrefix)
				if err != nil {
					return "", nil, err
				}
				condPart := extractConditionPart(subPath)
				if condPart != "" {
					// Remap variables
					for k, v := range subVars {
						newKey := fmt.Sprintf("v%d", varIdx)
						condPart = strings.Replace(condPart, "$"+k, "$"+newKey, 1)
						vars[newKey] = v
						varIdx++
					}
					orParts = append(orParts, condPart)
				}
			}
			if len(orParts) > 0 {
				allConditions = append(allConditions, "("+strings.Join(orParts, " || ")+")")
			}

		case "NOT":
			// Handle NOT: single filter map, negate
			notMap, ok := opMapRaw.(map[string]any)
			if !ok {
				return "", nil, fmt.Errorf("NOT must be a map")
			}
			subPath, subVars, err := buildJsonFilterFromOperatorMapWithPrefix(notMap, pathPrefix)
			if err != nil {
				return "", nil, err
			}
			condPart := extractConditionPart(subPath)
			if condPart != "" {
				// Remap variables
				for k, v := range subVars {
					newKey := fmt.Sprintf("v%d", varIdx)
					condPart = strings.Replace(condPart, "$"+k, "$"+newKey, 1)
					vars[newKey] = v
					varIdx++
				}
				allConditions = append(allConditions, "!("+condPart+")")
			}

		default:
			// Field with either operators or nested object/array filter
			opMap, ok := opMapRaw.(map[string]any)
			if !ok {
				return "", nil, fmt.Errorf("field %s value must be a map", field)
			}

			// Validate the field path
			if err := ValidatePath(field); err != nil {
				return "", nil, err
			}

			fullPath := pathPrefix + field

			// Check if this is an operator map or a nested filter
			if isOperatorMap(opMap) {
				// Process operators for this field
				conditions, fieldVars, err := processFieldOperators(fullPath, opMap, varIdx)
				if err != nil {
					return "", nil, err
				}
				allConditions = append(allConditions, conditions...)
				for k, v := range fieldVars {
					vars[k] = v
				}
				varIdx += len(fieldVars)
			} else {
				// Nested object filter - recurse with updated path prefix
				subPath, subVars, err := buildJsonFilterFromOperatorMapWithPrefix(opMap, fullPath+".")
				if err != nil {
					return "", nil, err
				}
				condPart := extractConditionPart(subPath)
				if condPart != "" {
					// Remap variables
					for k, v := range subVars {
						newKey := fmt.Sprintf("v%d", varIdx)
						condPart = strings.Replace(condPart, "$"+k, "$"+newKey, 1)
						vars[newKey] = v
						varIdx++
					}
					allConditions = append(allConditions, condPart)
				}
			}
		}
	}

	if len(allConditions) == 0 {
		return "", nil, fmt.Errorf("no valid conditions found")
	}

	jsonPath := fmt.Sprintf("$ ? (%s)", strings.Join(allConditions, " && "))
	return jsonPath, vars, nil
}

// processFieldOperators processes operators for a single field
func processFieldOperators(fieldPath string, opMap map[string]any, startVarIdx int) ([]string, map[string]any, error) {
	conditions := []string{}
	vars := make(map[string]any)
	varIdx := startVarIdx

	// Sort operators for deterministic output
	opKeys := make([]string, 0, len(opMap))
	for op := range opMap {
		opKeys = append(opKeys, op)
	}
	slices.Sort(opKeys)

	for _, op := range opKeys {
		value := opMap[op]
		varName := fmt.Sprintf("v%d", varIdx)

		switch op {
		case "isNull":
			isNull := cast.ToBool(value)
			if isNull {
				conditions = append(conditions, fmt.Sprintf("@.%s == null", fieldPath))
			} else {
				conditions = append(conditions, fmt.Sprintf("@.%s != null", fieldPath))
			}

		case "any":
			// Array filter: any element matches the condition
			// {"items": {"any": {"name": {"eq": "widget"}}}}
			// Generates: @.items[*].name == $v0 (for simple case)
			// Or for complex: exists(@.items[*] ? (@.name == $v0))
			anyFilter, ok := value.(map[string]any)
			if !ok {
				return nil, nil, fmt.Errorf("'any' operator value must be a map")
			}
			subPath, subVars, err := buildJsonFilterFromOperatorMapWithPrefix(anyFilter, "")
			if err != nil {
				return nil, nil, fmt.Errorf("processing 'any' filter: %w", err)
			}
			condPart := extractConditionPart(subPath)
			if condPart != "" {
				// Remap variables and replace @. with @.fieldPath[*].
				for k, v := range subVars {
					newKey := fmt.Sprintf("v%d", varIdx)
					condPart = strings.Replace(condPart, "$"+k, "$"+newKey, 1)
					vars[newKey] = v
					varIdx++
				}
				// Replace @. with @.fieldPath[*]. for array element access
				condPart = strings.ReplaceAll(condPart, "@.", fmt.Sprintf("@.%s[*].", fieldPath))
				conditions = append(conditions, condPart)
			}

		case "all":
			// Array filter: all elements match the condition
			// This is more complex in JSONPath - we check that no element fails
			// !(exists(@.items[*] ? (!(@.condition))))
			allFilter, ok := value.(map[string]any)
			if !ok {
				return nil, nil, fmt.Errorf("'all' operator value must be a map")
			}
			subPath, subVars, err := buildJsonFilterFromOperatorMapWithPrefix(allFilter, "")
			if err != nil {
				return nil, nil, fmt.Errorf("processing 'all' filter: %w", err)
			}
			condPart := extractConditionPart(subPath)
			if condPart != "" {
				// Remap variables
				for k, v := range subVars {
					newKey := fmt.Sprintf("v%d", varIdx)
					condPart = strings.Replace(condPart, "$"+k, "$"+newKey, 1)
					vars[newKey] = v
					varIdx++
				}
				// For 'all', we need: all elements in array satisfy condition
				// JSONPath doesn't have direct 'all' - we approximate with checking the condition
				condPart = strings.ReplaceAll(condPart, "@.", fmt.Sprintf("@.%s[*].", fieldPath))
				conditions = append(conditions, condPart)
			}

		default:
			jpOp, err := toJsonPathOp(op)
			if err != nil {
				return nil, nil, fmt.Errorf("field %s: %w", fieldPath, err)
			}

			conditions = append(conditions, fmt.Sprintf("@.%s %s $%s", fieldPath, jpOp, varName))
			vars[varName] = value
			varIdx++
		}
	}

	return conditions, vars, nil
}

// extractConditionPart extracts the condition from "$ ? (condition)"
func extractConditionPart(jsonPath string) string {
	// Remove "$ ? (" prefix and ")" suffix
	if strings.HasPrefix(jsonPath, "$ ? (") && strings.HasSuffix(jsonPath, ")") {
		return jsonPath[5 : len(jsonPath)-1]
	}
	return jsonPath
}

// BuildContainsExpression builds a PostgreSQL @> containment expression
// col @> '{"key": "val"}'::jsonb
func BuildContainsExpression(col exp.IdentifierExpression, value map[string]any) (goqu.Expression, error) {
	if len(value) == 0 {
		return nil, fmt.Errorf("contains value cannot be empty")
	}

	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal contains value: %w", err)
	}

	// Use literal SQL for @> operator: col @> 'json'::jsonb
	return goqu.L("? @> ?::jsonb", col, string(jsonBytes)), nil
}

// BuildJsonPathExistsExpression builds a PostgreSQL jsonb_path_exists expression
// jsonb_path_exists(col, 'jsonpath'::jsonpath, 'vars'::jsonb)
func BuildJsonPathExistsExpression(col exp.IdentifierExpression, jsonPath string, vars map[string]any) (goqu.Expression, error) {
	if jsonPath == "" {
		return nil, fmt.Errorf("jsonPath cannot be empty")
	}

	if len(vars) == 0 {
		// No variables, simpler form
		return goqu.L("jsonb_path_exists(?, ?::jsonpath)", col, jsonPath), nil
	}

	varsJson, err := json.Marshal(vars)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal vars: %w", err)
	}

	return goqu.L("jsonb_path_exists(?, ?::jsonpath, ?::jsonb)", col, jsonPath, string(varsJson)), nil
}

// BuildMapFilter builds goqu expressions for a JsonFilter (Map scalar filtering)
func BuildMapFilter(col exp.IdentifierExpression, filter JsonFilter) (goqu.Expression, error) {
	expList := exp.NewExpressionList(exp.AndType)

	// Handle isNull
	if filter.IsNull != nil {
		if *filter.IsNull {
			expList = expList.Append(col.IsNull())
		} else {
			expList = expList.Append(col.IsNotNull())
		}
	}

	// Handle contains (@>)
	if len(filter.Contains) > 0 {
		containsExp, err := BuildContainsExpression(col, filter.Contains)
		if err != nil {
			return nil, err
		}
		expList = expList.Append(containsExp)
	}

	// Handle where (AND conditions)
	if len(filter.Where) > 0 {
		jsonPath, vars, err := BuildJsonPathExpression(filter.Where, "AND")
		if err != nil {
			return nil, fmt.Errorf("building where conditions: %w", err)
		}
		pathExp, err := BuildJsonPathExistsExpression(col, jsonPath, vars)
		if err != nil {
			return nil, err
		}
		expList = expList.Append(pathExp)
	}

	// Handle whereAny (OR conditions)
	if len(filter.WhereAny) > 0 {
		jsonPath, vars, err := BuildJsonPathExpression(filter.WhereAny, "OR")
		if err != nil {
			return nil, fmt.Errorf("building whereAny conditions: %w", err)
		}
		pathExp, err := BuildJsonPathExistsExpression(col, jsonPath, vars)
		if err != nil {
			return nil, err
		}
		expList = expList.Append(pathExp)
	}

	return expList, nil
}

// ParseMapComparator parses a map[string]any into a JsonFilter struct
func ParseMapComparator(filterMap map[string]any) (JsonFilter, error) {
	var filter JsonFilter

	if contains, ok := filterMap["contains"].(map[string]any); ok {
		filter.Contains = contains
	}

	if isNull, ok := filterMap["isNull"]; ok {
		b := cast.ToBool(isNull)
		filter.IsNull = &b
	}

	if where, ok := filterMap["where"].([]any); ok {
		conditions, err := parsePathConditions(where)
		if err != nil {
			return filter, fmt.Errorf("parsing where: %w", err)
		}
		filter.Where = conditions
	}

	if whereAny, ok := filterMap["whereAny"].([]any); ok {
		conditions, err := parsePathConditions(whereAny)
		if err != nil {
			return filter, fmt.Errorf("parsing whereAny: %w", err)
		}
		filter.WhereAny = conditions
	}

	return filter, nil
}

// parsePathConditions parses an array of condition maps into JsonPathCondition slice
func parsePathConditions(conditions []any) ([]JsonPathCondition, error) {
	result := make([]JsonPathCondition, 0, len(conditions))

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

			result = append(result, JsonPathCondition{
				Path:  path,
				Op:    op,
				Value: value,
			})
		}
	}

	return result, nil
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
			if err := ValidatePath(sel.Name); err != nil {
				return nil, fmt.Errorf("invalid JSON field name %s: %w", sel.Name, err)
			}
			// Build path using -> operator: col->'field' for JSONB, or col->>'field' for text
			// We use -> to get JSONB, which works well with jsonb_build_object
			valueExpr = goqu.L("?->?", baseCol, sel.Name)

		case builders.TypeObject, builders.TypeJson:
			// Nested object: extract the nested JSON object first, then recursively build
			if err := ValidatePath(sel.Name); err != nil {
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
