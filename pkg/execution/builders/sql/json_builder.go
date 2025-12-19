package sql

import (
	"fmt"

	"github.com/doug-martin/goqu/v9/exp"
)

// JSONFilterBuilder provides a fluent API for building JSON filter expressions
type JSONFilterBuilder struct {
	column     exp.IdentifierExpression
	dialect    Dialect
	exprs      []exp.Expression
	currentAnd []exp.Expression // Accumulator for AND conditions
	currentOr  []exp.Expression // Accumulator for OR conditions
}

// NewJSONFilterBuilder creates a new JSON filter builder
func NewJSONFilterBuilder(col exp.IdentifierExpression, dialect Dialect) *JSONFilterBuilder {
	return &JSONFilterBuilder{
		column:     col,
		dialect:    dialect,
		exprs:      make([]exp.Expression, 0),
		currentAnd: make([]exp.Expression, 0),
		currentOr:  make([]exp.Expression, 0),
	}
}

// Where adds a condition with a specific operator
func (b *JSONFilterBuilder) Where(path string, operator string, value any) *JSONFilterBuilder {
	cond, err := NewJSONPathCondition(path, operator, value)
	if err != nil {
		// Store error for Build() to handle
		b.exprs = append(b.exprs, nil)
		return b
	}

	// Create a filter with this single condition
	filter := NewJSONPathFilter(b.column, b.dialect)
	filter.AddCondition(cond)
	expr, err := filter.Expression()
	if err != nil {
		b.exprs = append(b.exprs, nil)
		return b
	}

	b.currentAnd = append(b.currentAnd, expr)
	return b
}

// WhereNull adds a NULL check condition
func (b *JSONFilterBuilder) WhereNull(path string) *JSONFilterBuilder {
	return b.Where(path, "isNull", true)
}

// WhereNotNull adds a NOT NULL check condition
func (b *JSONFilterBuilder) WhereNotNull(path string) *JSONFilterBuilder {
	return b.Where(path, "isNull", false)
}

// Eq is a convenience method for equality
func (b *JSONFilterBuilder) Eq(path string, value any) *JSONFilterBuilder {
	return b.Where(path, "eq", value)
}

// Neq is a convenience method for inequality
func (b *JSONFilterBuilder) Neq(path string, value any) *JSONFilterBuilder {
	return b.Where(path, "neq", value)
}

// Gt is a convenience method for greater than
func (b *JSONFilterBuilder) Gt(path string, value any) *JSONFilterBuilder {
	return b.Where(path, "gt", value)
}

// Gte is a convenience method for greater than or equal
func (b *JSONFilterBuilder) Gte(path string, value any) *JSONFilterBuilder {
	return b.Where(path, "gte", value)
}

// Lt is a convenience method for less than
func (b *JSONFilterBuilder) Lt(path string, value any) *JSONFilterBuilder {
	return b.Where(path, "lt", value)
}

// Lte is a convenience method for less than or equal
func (b *JSONFilterBuilder) Lte(path string, value any) *JSONFilterBuilder {
	return b.Where(path, "lte", value)
}

// Like is a convenience method for pattern matching
func (b *JSONFilterBuilder) Like(path string, pattern string) *JSONFilterBuilder {
	return b.Where(path, "like", pattern)
}

// Contains adds a containment check (@> operator)
func (b *JSONFilterBuilder) Contains(value map[string]any) *JSONFilterBuilder {
	containsExpr := NewJSONContains(b.column, value, b.dialect)
	expr, err := containsExpr.Expression()
	if err != nil {
		b.exprs = append(b.exprs, nil)
		return b
	}

	b.currentAnd = append(b.currentAnd, expr)
	return b
}

// Or flushes current AND conditions and starts a new OR group
func (b *JSONFilterBuilder) Or() *JSONFilterBuilder {
	// Flush current AND conditions
	if len(b.currentAnd) > 0 {
		if len(b.currentAnd) == 1 {
			b.currentOr = append(b.currentOr, b.currentAnd[0])
		} else {
			andExpr := NewJSONLogicalExpr(exp.AndType)
			for _, expr := range b.currentAnd {
				andExpr.AddExpression(expr)
			}
			combined, err := andExpr.Expression()
			if err == nil {
				b.currentOr = append(b.currentOr, combined)
			}
		}
		b.currentAnd = make([]exp.Expression, 0)
	}
	return b
}

// Not wraps a set of conditions in a NOT operator
func (b *JSONFilterBuilder) Not(conditionFn func(*JSONFilterBuilder)) *JSONFilterBuilder {
	// Create a sub-builder
	subBuilder := NewJSONFilterBuilder(b.column, b.dialect)
	conditionFn(subBuilder)

	// Build the sub-expression
	subExpr, err := subBuilder.Build()
	if err != nil {
		b.exprs = append(b.exprs, nil)
		return b
	}

	// Wrap in NOT
	notExpr := NewJSONLogicalExpr(exp.AndType)
	notExpr.AddExpression(subExpr)
	notExpr.SetNegate(true)

	combined, err := notExpr.Expression()
	if err != nil {
		b.exprs = append(b.exprs, nil)
		return b
	}

	b.currentAnd = append(b.currentAnd, combined)
	return b
}

// IsNull adds a NULL check on the column itself
func (b *JSONFilterBuilder) IsNull(isNull bool) *JSONFilterBuilder {
	nullCheck := NewJSONNullCheck(b.column, isNull, b.dialect)
	expr, err := nullCheck.Expression()
	if err != nil {
		b.exprs = append(b.exprs, nil)
		return b
	}

	b.currentAnd = append(b.currentAnd, expr)
	return b
}

// Build finalizes and returns the expression
func (b *JSONFilterBuilder) Build() (exp.Expression, error) {
	// Flush any remaining AND conditions
	if len(b.currentAnd) > 0 {
		if len(b.currentAnd) == 1 {
			b.currentOr = append(b.currentOr, b.currentAnd[0])
		} else {
			andExpr := NewJSONLogicalExpr(exp.AndType)
			for _, expr := range b.currentAnd {
				if expr == nil {
					return nil, fmt.Errorf("invalid expression in builder")
				}
				andExpr.AddExpression(expr)
			}
			combined, err := andExpr.Expression()
			if err != nil {
				return nil, err
			}
			b.currentOr = append(b.currentOr, combined)
		}
	}

	// Combine OR expressions
	if len(b.currentOr) == 0 && len(b.exprs) == 0 {
		return nil, fmt.Errorf("no conditions to build")
	}

	if len(b.currentOr) == 1 {
		return b.currentOr[0], nil
	}

	if len(b.currentOr) > 1 {
		orExpr := NewJSONLogicalExpr(exp.OrType)
		for _, expr := range b.currentOr {
			if expr == nil {
				return nil, fmt.Errorf("invalid expression in OR group")
			}
			orExpr.AddExpression(expr)
		}
		return orExpr.Expression()
	}

	// Fallback to exprs if no OR groups
	if len(b.exprs) == 1 {
		if b.exprs[0] == nil {
			return nil, fmt.Errorf("invalid expression")
		}
		return b.exprs[0], nil
	}

	andExpr := NewJSONLogicalExpr(exp.AndType)
	for _, expr := range b.exprs {
		if expr == nil {
			return nil, fmt.Errorf("invalid expression in builder")
		}
		andExpr.AddExpression(expr)
	}
	return andExpr.Expression()
}

// Helper method for creating simple filters
// This is used by the conversion layer

// BuildSimpleFilter creates a filter with a single condition
func BuildSimpleFilter(col exp.IdentifierExpression, path string, operator string, value any, dialect Dialect) (exp.Expression, error) {
	return NewJSONFilterBuilder(col, dialect).
		Where(path, operator, value).
		Build()
}

// BuildLogicalFilter creates a filter with AND/OR/NOT logic
func BuildLogicalFilter(col exp.IdentifierExpression, logic exp.ExpressionListType, exprs []exp.Expression, negate bool) (exp.Expression, error) {
	logicalExpr := NewJSONLogicalExpr(logic)
	for _, expr := range exprs {
		logicalExpr.AddExpression(expr)
	}
	logicalExpr.SetNegate(negate)
	return logicalExpr.Expression()
}
