package sql

import (
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9/exp"
)

// JSONPathExpr is a marker interface for JSONPath expression nodes
type JSONPathExpr interface {
	jsonPathExpr()
}

// ConditionExpr represents a single condition: @.path op $var
type ConditionExpr struct {
	Path   string
	Op     JSONPathOp
	Value  any
	IsNull *bool  // nil = not null check, true/false for null checks
	Regex  string // for like_regex (value embedded)
}

func (ConditionExpr) jsonPathExpr() {}

// LogicalExpr combines expressions with AND/OR
type LogicalExpr struct {
	Op       JSONPathOp // JSONPathAnd or JSONPathOr
	Children []JSONPathExpr
	Negate   bool
}

func (LogicalExpr) jsonPathExpr() {}

// Expression constructors - goqu style

// JsonExpr creates a condition expression from path, operator, and value
func JsonExpr(path, op string, value any) (JSONPathExpr, error) {
	return newCondition(path, op, value)
}

// JsonOr combines expressions with OR
func JsonOr(exprs ...JSONPathExpr) JSONPathExpr {
	return &LogicalExpr{Op: JSONPathOr, Children: exprs}
}

// JsonAnd combines expressions with AND
func JsonAnd(exprs ...JSONPathExpr) JSONPathExpr {
	return &LogicalExpr{Op: JSONPathAnd, Children: exprs}
}

// JsonNot negates an expression
func JsonNot(expr JSONPathExpr) JSONPathExpr {
	return &LogicalExpr{Op: JSONPathAnd, Children: []JSONPathExpr{expr}, Negate: true}
}

// JsonAny creates an array "any" filter - matches if any element satisfies conditions
func JsonAny(arrayPath string, exprs ...JSONPathExpr) JSONPathExpr {
	inner := &LogicalExpr{Op: JSONPathAnd, Children: exprs}
	return convertToArrayExpr(arrayPath, inner, false)
}

// JsonAll creates an array "all" filter - matches if all elements satisfy conditions
func JsonAll(arrayPath string, exprs ...JSONPathExpr) JSONPathExpr {
	inner := &LogicalExpr{Op: JSONPathAnd, Children: exprs}
	return convertToArrayExpr(arrayPath, inner, true)
}

// JSONPathBuilder walks expression tree and produces JSONPath string + vars
type JSONPathBuilder struct {
	vars   map[string]any
	offset int
}

// NewJSONPathBuilder creates a new builder
func NewJSONPathBuilder() *JSONPathBuilder {
	return &JSONPathBuilder{
		vars:   make(map[string]any),
		offset: 0,
	}
}

// Build converts an expression tree to a JSONPath condition string
func (b *JSONPathBuilder) Build(expr JSONPathExpr) string {
	switch e := expr.(type) {
	case *ConditionExpr:
		return b.buildCondition(e)
	case *LogicalExpr:
		return b.buildLogical(e)
	default:
		return ""
	}
}

func (b *JSONPathBuilder) buildCondition(c *ConditionExpr) string {
	if c.IsNull != nil {
		if *c.IsNull {
			return fmt.Sprintf("@.%s == null", c.Path)
		}
		return fmt.Sprintf("@.%s != null", c.Path)
	}

	if c.Regex != "" {
		return fmt.Sprintf("@.%s %s %s", c.Path, JSONPathLikeRegex, c.Regex)
	}

	varName := fmt.Sprintf("v%d", b.offset)
	b.vars[varName] = c.Value
	b.offset++
	return fmt.Sprintf("@.%s %s $%s", c.Path, c.Op, varName)
}

func (b *JSONPathBuilder) buildLogical(l *LogicalExpr) string {
	if len(l.Children) == 0 {
		return ""
	}

	parts := make([]string, 0, len(l.Children))
	for _, child := range l.Children {
		if part := b.Build(child); part != "" {
			parts = append(parts, part)
		}
	}

	if len(parts) == 0 {
		return ""
	}

	result := strings.Join(parts, fmt.Sprintf(" %s ", l.Op))

	if l.Op == JSONPathOr && len(parts) > 1 {
		result = "(" + result + ")"
	}

	if l.Negate {
		result = "!(" + result + ")"
	}

	return result
}

// Vars returns the collected variables
func (b *JSONPathBuilder) Vars() map[string]any {
	return b.vars
}

// JSONFilterBuilder builds JSON filter expressions.
// All conditions are combined into a single jsonb_path_exists() call.
type JSONFilterBuilder struct {
	column  exp.IdentifierExpression
	dialect Dialect
	exprs   []JSONPathExpr
	err     error
}

// NewJSONFilterBuilder creates a new JSON filter builder
func NewJSONFilterBuilder(col exp.IdentifierExpression, dialect Dialect) *JSONFilterBuilder {
	return &JSONFilterBuilder{
		column:  col,
		dialect: dialect,
		exprs:   make([]JSONPathExpr, 0),
	}
}

// Where adds one or more expressions (AND'd together by default)
func (b *JSONFilterBuilder) Where(exprs ...JSONPathExpr) *JSONFilterBuilder {
	if b.err != nil {
		return b
	}
	b.exprs = append(b.exprs, exprs...)
	return b
}

// WhereOp adds a condition with path, operator, and value
func (b *JSONFilterBuilder) WhereOp(path, op string, value any) *JSONFilterBuilder {
	if b.err != nil {
		return b
	}
	cond, err := newCondition(path, op, value)
	if err != nil {
		b.err = err
		return b
	}
	b.exprs = append(b.exprs, cond)
	return b
}

// Eq adds an equality condition
func (b *JSONFilterBuilder) Eq(path string, value any) *JSONFilterBuilder {
	return b.WhereOp(path, "eq", value)
}

// Neq adds an inequality condition
func (b *JSONFilterBuilder) Neq(path string, value any) *JSONFilterBuilder {
	return b.WhereOp(path, "neq", value)
}

// Gt adds a greater-than condition
func (b *JSONFilterBuilder) Gt(path string, value any) *JSONFilterBuilder {
	return b.WhereOp(path, "gt", value)
}

// Gte adds a greater-than-or-equal condition
func (b *JSONFilterBuilder) Gte(path string, value any) *JSONFilterBuilder {
	return b.WhereOp(path, "gte", value)
}

// Lt adds a less-than condition
func (b *JSONFilterBuilder) Lt(path string, value any) *JSONFilterBuilder {
	return b.WhereOp(path, "lt", value)
}

// Lte adds a less-than-or-equal condition
func (b *JSONFilterBuilder) Lte(path string, value any) *JSONFilterBuilder {
	return b.WhereOp(path, "lte", value)
}

// Like adds a regex pattern match condition
func (b *JSONFilterBuilder) Like(path string, pattern string) *JSONFilterBuilder {
	return b.WhereOp(path, "like", pattern)
}

// Prefix adds a prefix match condition
func (b *JSONFilterBuilder) Prefix(path string, prefix string) *JSONFilterBuilder {
	return b.WhereOp(path, "prefix", prefix)
}

// Suffix adds a suffix match condition
func (b *JSONFilterBuilder) Suffix(path string, suffix string) *JSONFilterBuilder {
	return b.WhereOp(path, "suffix", suffix)
}

// Contains adds a substring match condition
func (b *JSONFilterBuilder) Contains(path string, substr string) *JSONFilterBuilder {
	return b.WhereOp(path, "contains", substr)
}

// ILike adds a case-insensitive pattern match condition
func (b *JSONFilterBuilder) ILike(path string, pattern string) *JSONFilterBuilder {
	return b.WhereOp(path, "ilike", pattern)
}

// IsNull adds a null check condition
func (b *JSONFilterBuilder) IsNull(path string) *JSONFilterBuilder {
	return b.WhereOp(path, "isNull", true)
}

// IsNotNull adds a not-null check condition
func (b *JSONFilterBuilder) IsNotNull(path string) *JSONFilterBuilder {
	return b.WhereOp(path, "isNull", false)
}

// Build finalizes and returns the goqu expression
func (b *JSONFilterBuilder) Build() (exp.Expression, error) {
	if b.err != nil {
		return nil, b.err
	}

	if len(b.exprs) == 0 {
		return nil, fmt.Errorf("no conditions to build")
	}

	// Wrap all expressions in AND
	root := &LogicalExpr{Op: JSONPathAnd, Children: b.exprs}

	builder := NewJSONPathBuilder()
	condStr := builder.Build(root)

	if condStr == "" {
		return nil, fmt.Errorf("no valid conditions")
	}

	jsonPath := fmt.Sprintf("$ ? (%s)", condStr)
	return b.dialect.JSONPathExists(b.column, jsonPath, builder.Vars()), nil
}

func convertToArrayExpr(arrayPath string, expr *LogicalExpr, invert bool) JSONPathExpr {
	children := make([]JSONPathExpr, 0, len(expr.Children))

	for _, child := range expr.Children {
		switch c := child.(type) {
		case *ConditionExpr:
			newPath := arrayPath + "[*]"
			if c.Path != "" {
				newPath = arrayPath + "[*]." + c.Path
			}

			newCond := &ConditionExpr{
				Path:   newPath,
				Op:     c.Op,
				Value:  c.Value,
				IsNull: c.IsNull,
				Regex:  c.Regex,
			}

			if invert && c.Op != "" {
				if inverted, ok := invertedJSONPathOp[c.Op]; ok {
					newCond.Op = inverted
				}
			}

			children = append(children, newCond)

		case *LogicalExpr:
			children = append(children, convertToArrayExpr(arrayPath, c, invert))
		}
	}

	return &LogicalExpr{
		Op:       expr.Op,
		Children: children,
		Negate:   invert,
	}
}

// newCondition creates a ConditionExpr for a field and operator
func newCondition(fieldPath, op string, value any) (*ConditionExpr, error) {
	switch op {
	case "isNull":
		isNull, ok := value.(bool)
		if !ok {
			return nil, fmt.Errorf("isNull value must be a boolean")
		}
		return &ConditionExpr{Path: fieldPath, IsNull: &isNull}, nil

	case "prefix":
		strVal, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("prefix value must be a string")
		}
		return &ConditionExpr{Path: fieldPath, Regex: `"^` + escapeRegexPattern(strVal) + `"`}, nil

	case "suffix":
		strVal, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("suffix value must be a string")
		}
		return &ConditionExpr{Path: fieldPath, Regex: `"` + escapeRegexPattern(strVal) + `$"`}, nil

	case "ilike":
		strVal, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("ilike value must be a string")
		}
		return &ConditionExpr{Path: fieldPath, Regex: `"` + escapeRegexPattern(strVal) + `" flag "i"`}, nil

	case "contains":
		strVal, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("contains value must be a string")
		}
		return &ConditionExpr{Path: fieldPath, Regex: `"` + escapeRegexPattern(strVal) + `"`}, nil

	default:
		jpOp, ok := graphqlToJSONPathOp[op]
		if !ok {
			return nil, fmt.Errorf("unsupported operator: %s", op)
		}
		return &ConditionExpr{Path: fieldPath, Op: jpOp, Value: value}, nil
	}
}
