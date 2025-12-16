package sql

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
)

// Enhanced path validation regex - supports multiple array indices
// Allows: field, field.nested, field[0], field[0][1], field[0].nested[1][2], etc.
var pathValidationRegexV2 = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*(\[[0-9]+\])*(\.[a-zA-Z_][a-zA-Z0-9_]*(\[[0-9]+\])*)*$`)

// ValidatePathV2 validates a JSON path with support for multiple array indices
func ValidatePathV2(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}
	if !pathValidationRegexV2.MatchString(path) {
		return fmt.Errorf("invalid path format: %s", path)
	}
	return nil
}

// LogicType represents the logical combination type for conditions
type LogicType int

const (
	LogicAnd LogicType = iota
	LogicOr
)

// ArrayFilterMode represents how array filtering should work
type ArrayFilterMode int

const (
	ArrayAny ArrayFilterMode = iota
	ArrayAll
)

// JSONPathConditionExpr represents a single JSONPath condition
// This is an expression builder that produces a JSONPath condition string
type JSONPathConditionExpr struct {
	path     string
	operator string
	value    any
	varName  string // Variable name for parameterization
}

// NewJSONPathCondition creates a new JSON path condition
func NewJSONPathCondition(path string, operator string, value any) (*JSONPathConditionExpr, error) {
	if err := ValidatePathV2(path); err != nil {
		return nil, err
	}

	return &JSONPathConditionExpr{
		path:     path,
		operator: operator,
		value:    value,
	}, nil
}

// SetVarName sets the variable name for this condition
func (j *JSONPathConditionExpr) SetVarName(name string) {
	j.varName = name
}

// ToJSONPathString converts the condition to a JSONPath condition string
// Returns the condition part (e.g., "@.field == $v0") and the value
func (j *JSONPathConditionExpr) ToJSONPathString() (string, any, error) {
	// Handle isNull specially
	if j.operator == "isNull" {
		isNull, ok := j.value.(bool)
		if !ok {
			// Try to convert
			isNull = fmt.Sprintf("%v", j.value) == "true"
		}
		if isNull {
			return fmt.Sprintf("@.%s == null", j.path), nil, nil
		}
		return fmt.Sprintf("@.%s != null", j.path), nil, nil
	}

	// Map operator to JSONPath operator
	jpOp, err := toJsonPathOp(j.operator)
	if err != nil {
		return "", nil, err
	}

	// Build condition with variable reference
	if j.varName == "" {
		j.varName = "v0" // Default if not set
	}

	condition := fmt.Sprintf("@.%s %s $%s", j.path, jpOp, j.varName)
	return condition, j.value, nil
}

// JSONPathFilterExpr combines multiple conditions into a single JSONPath filter
type JSONPathFilterExpr struct {
	column      exp.IdentifierExpression
	conditions  []*JSONPathConditionExpr
	logic       LogicType
	dialect     Dialect
	wrapORInPar bool // When true, wrap OR conditions in extra parentheses (for logical OR operator compatibility)
	negate      bool // When true, negate the entire condition in JSONPath using ! operator
}

// NewJSONPathFilter creates a new JSONPath filter expression
func NewJSONPathFilter(col exp.IdentifierExpression, dialect Dialect) *JSONPathFilterExpr {
	return &JSONPathFilterExpr{
		column:     col,
		conditions: make([]*JSONPathConditionExpr, 0),
		logic:      LogicAnd,
		dialect:    dialect,
	}
}

// AddCondition adds a condition to the filter
func (j *JSONPathFilterExpr) AddCondition(cond *JSONPathConditionExpr) {
	j.conditions = append(j.conditions, cond)
}

// SetLogic sets the logic type (AND/OR) for combining conditions
func (j *JSONPathFilterExpr) SetLogic(logic LogicType) {
	j.logic = logic
}

// SetNegate sets whether to negate the entire condition in JSONPath
func (j *JSONPathFilterExpr) SetNegate(negate bool) {
	j.negate = negate
}

// Expression builds the final goqu expression
func (j *JSONPathFilterExpr) Expression() (exp.Expression, error) {
	if len(j.conditions) == 0 {
		return nil, fmt.Errorf("no conditions to build filter from")
	}

	// Build JSONPath and collect variables
	vars := make(map[string]any)
	conditionParts := make([]string, 0, len(j.conditions))

	for i, cond := range j.conditions {
		// Set variable name
		varName := fmt.Sprintf("v%d", i)
		cond.SetVarName(varName)

		// Get condition string
		condStr, val, err := cond.ToJSONPathString()
		if err != nil {
			return nil, err
		}

		conditionParts = append(conditionParts, condStr)
		if val != nil {
			vars[varName] = val
		}
	}

	// Combine conditions with logic operator
	connector := " && "
	if j.logic == LogicOr {
		connector = " || "
	}

	// Build final JSONPath
	var conditionStr string
	if len(conditionParts) == 1 {
		conditionStr = conditionParts[0]
	} else {
		combinedConditions := ""
		for i, part := range conditionParts {
			if i > 0 {
				combinedConditions += connector
			}
			combinedConditions += part
		}
		// For OR logic with multiple conditions, add extra parentheses when requested (for logical OR operator)
		if j.logic == LogicOr && j.wrapORInPar {
			conditionStr = fmt.Sprintf("(%s)", combinedConditions)
		} else {
			conditionStr = combinedConditions
		}
	}

	// Apply negation if requested (negate inside JSONPath, not SQL wrapper)
	if j.negate {
		conditionStr = fmt.Sprintf("!(%s)", conditionStr)
	}

	jsonPath := fmt.Sprintf("$ ? (%s)", conditionStr)

	// Use dialect to build the expression
	return j.dialect.JSONPathExists(j.column, jsonPath, vars), nil
}

// JSONContainsExpr represents a JSON containment check (@> operator)
type JSONContainsExpr struct {
	column  exp.IdentifierExpression
	value   map[string]any
	dialect Dialect
}

// NewJSONContains creates a new JSON contains expression
func NewJSONContains(col exp.IdentifierExpression, value map[string]any, dialect Dialect) *JSONContainsExpr {
	return &JSONContainsExpr{
		column:  col,
		value:   value,
		dialect: dialect,
	}
}

// Expression builds the final goqu expression
func (j *JSONContainsExpr) Expression() (exp.Expression, error) {
	if len(j.value) == 0 {
		return nil, fmt.Errorf("contains value cannot be empty")
	}

	jsonBytes, err := json.Marshal(j.value)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal contains value: %w", err)
	}

	return j.dialect.JSONContains(j.column, string(jsonBytes)), nil
}

// JSONArrayFilterExpr represents array filtering with any/all operators
type JSONArrayFilterExpr struct {
	column     exp.IdentifierExpression
	arrayPath  string
	conditions []*JSONPathConditionExpr
	mode       ArrayFilterMode
	dialect    Dialect
}

// NewJSONArrayFilter creates a new array filter expression
func NewJSONArrayFilter(col exp.IdentifierExpression, arrayPath string, mode ArrayFilterMode, dialect Dialect) (*JSONArrayFilterExpr, error) {
	if err := ValidatePathV2(arrayPath); err != nil {
		return nil, err
	}

	return &JSONArrayFilterExpr{
		column:     col,
		arrayPath:  arrayPath,
		conditions: make([]*JSONPathConditionExpr, 0),
		mode:       mode,
		dialect:    dialect,
	}, nil
}

// AddCondition adds a condition for array elements
func (j *JSONArrayFilterExpr) AddCondition(cond *JSONPathConditionExpr) {
	j.conditions = append(j.conditions, cond)
}

// Expression builds the final goqu expression
func (j *JSONArrayFilterExpr) Expression() (exp.Expression, error) {
	if len(j.conditions) == 0 {
		return nil, fmt.Errorf("no conditions for array filter")
	}

	// Build conditions
	vars := make(map[string]any)
	conditionParts := make([]string, 0, len(j.conditions))

	for i, cond := range j.conditions {
		varName := fmt.Sprintf("v%d", i)
		cond.SetVarName(varName)

		condStr, val, err := cond.ToJSONPathString()
		if err != nil {
			return nil, err
		}

		// Replace @. with @.arrayPath[*]. for array element access
		condStr = fmt.Sprintf("@.%s[*].%s", j.arrayPath, condStr[2:]) // Remove @. prefix

		conditionParts = append(conditionParts, condStr)
		if val != nil {
			vars[varName] = val
		}
	}

	// Combine conditions
	combinedConditions := ""
	if len(conditionParts) == 1 {
		combinedConditions = conditionParts[0]
	} else {
		for i, part := range conditionParts {
			if i > 0 {
				combinedConditions += " && "
			}
			combinedConditions += part
		}
	}

	var jsonPath string
	if j.mode == ArrayAny {
		// For 'any': at least one element matches
		jsonPath = fmt.Sprintf("$ ? (%s)", combinedConditions)
		return j.dialect.JSONPathExists(j.column, jsonPath, vars), nil
	} else {
		// For 'all': ALL elements must match the condition
		// PostgreSQL approach: Check that no element violates the condition
		// We need to negate the condition and check that it doesn't exist

		// Build the negated condition for each part
		negatedParts := make([]string, 0, len(conditionParts))
		for _, part := range conditionParts {
			// Negate the condition
			negatedParts = append(negatedParts, fmt.Sprintf("!(%s)", part))
		}

		negatedConditions := ""
		if len(negatedParts) == 1 {
			negatedConditions = negatedParts[0]
		} else {
			// For multiple conditions in 'all', we negate each AND combine with OR
			// Because: all(A && B) = !(exists(!A || !B))
			for i, part := range negatedParts {
				if i > 0 {
					negatedConditions += " || "
				}
				negatedConditions += part
			}
		}

		// Check that NO element matches the negated condition
		// jsonb_path_exists returns true if ANY element matches
		// So we use NOT jsonb_path_exists with the negated condition
		jsonPath = fmt.Sprintf("$ ? (%s)", negatedConditions)
		innerExpr := j.dialect.JSONPathExists(j.column, jsonPath, vars)

		// Wrap in NOT to get "no element violates the condition" = "all elements satisfy"
		return goqu.Func("NOT", innerExpr), nil
	}
}

// JSONNullCheckExpr represents a NULL check expression
type JSONNullCheckExpr struct {
	column  exp.IdentifierExpression
	isNull  bool
	dialect Dialect
}

// NewJSONNullCheck creates a new NULL check expression
func NewJSONNullCheck(col exp.IdentifierExpression, isNull bool, dialect Dialect) *JSONNullCheckExpr {
	return &JSONNullCheckExpr{
		column:  col,
		isNull:  isNull,
		dialect: dialect,
	}
}

// Expression builds the final goqu expression
func (j *JSONNullCheckExpr) Expression() (exp.Expression, error) {
	if j.isNull {
		return j.column.IsNull(), nil
	}
	return j.column.IsNotNull(), nil
}

// JSONLogicalExpr combines multiple expressions with AND/OR/NOT
type JSONLogicalExpr struct {
	expressions []exp.Expression
	logic       LogicType
	negate      bool
}

// NewJSONLogicalExpr creates a new logical expression combiner
func NewJSONLogicalExpr(logic LogicType) *JSONLogicalExpr {
	return &JSONLogicalExpr{
		expressions: make([]exp.Expression, 0),
		logic:       logic,
		negate:      false,
	}
}

// AddExpression adds an expression to the logical combination
func (j *JSONLogicalExpr) AddExpression(expr exp.Expression) {
	j.expressions = append(j.expressions, expr)
}

// SetNegate sets whether to negate the entire expression (NOT operator)
func (j *JSONLogicalExpr) SetNegate(negate bool) {
	j.negate = negate
}

// Expression builds the final goqu expression
func (j *JSONLogicalExpr) Expression() (exp.Expression, error) {
	if len(j.expressions) == 0 {
		return nil, fmt.Errorf("no expressions to combine")
	}

	// Build expression list
	var result exp.Expression
	if j.logic == LogicAnd {
		expList := exp.NewExpressionList(exp.AndType, j.expressions...)
		result = expList
	} else {
		expList := exp.NewExpressionList(exp.OrType, j.expressions...)
		result = expList
	}

	// Apply negation if requested
	if j.negate {
		result = goqu.Func("NOT", result)
	}

	return result, nil
}
