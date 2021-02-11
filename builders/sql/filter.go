package sql

import (
	"fmt"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/iancoleman/strcase"
	"github.com/roneli/fastgql/builders"
	"github.com/roneli/fastgql/schema"
	"github.com/spf13/cast"
	"github.com/vektah/gqlparser/v2/ast"
)

// expressionsBuilder allows to create internal an internal filterBuilder for nested filtering
type expressionsBuilder struct {
	exp.ExpressionList
	builder *Builder
}

func newExpressionBuilder(b *Builder, logicalType schema.LogicalOperator) *expressionsBuilder {
	switch logicalType {
	case schema.LogicalOperatorOR:
		return &expressionsBuilder{
			ExpressionList: exp.NewExpressionList(exp.OrType),
			builder:        b,
		}
	default:
		return &expressionsBuilder{
			ExpressionList: exp.NewExpressionList(exp.AndType),
			builder:        b,
		}
	}
}

func (e *expressionsBuilder) Config() *builders.Config {
	return e.builder.Config()
}

func (e *expressionsBuilder) Operation(name, key string, value interface{}) error {

	op, ok := e.builder.operators[key]
	if !ok {
		return fmt.Errorf("key operator %s not supported", key)
	}
	e.ExpressionList = e.Append(op(e.builder.table, strcase.ToSnake(name), value))
	return nil
}

func (e *expressionsBuilder) Filter(f *ast.FieldDefinition, key string, values map[string]interface{}) error {
	fieldDef := e.builder.Config().Schema.Types[f.Type.Name()]
	filterFieldDef := fieldDef.Fields.ForName(key)
	// Create a builder
	d := filterFieldDef.Directives.ForName("sqlRelation")
	if d == nil {
		return fmt.Errorf("missing directive sqlRelation")
	}
	fb, err := buildFilterInput(e.builder, parseRelationDirective(d))
	if err != nil {
		return err
	}

	if err := builders.BuildFilter(&fb, filterFieldDef, values); err != nil {
		return err
	}

	e.ExpressionList = e.Append(goqu.Func("exists", fb.builder))
	return nil
}

func (e *expressionsBuilder) Logical(f *ast.FieldDefinition, logicalExp schema.LogicalOperator, values []interface{}) error {
	expList := newExpressionBuilder(e.builder, logicalExp)
	for _, value := range values {
		v, ok := value.(map[string]interface{})
		if !ok {
			return fmt.Errorf("failed cast value %s", value)
		}
		if err := builders.BuildFilter(expList, f, v); err != nil {
			return err
		}
		e.ExpressionList = expList
	}
	return nil
}

type Operator func(table exp.AliasedExpression, key string, value interface{}) goqu.Expression

func Eq(table exp.AliasedExpression, key string, value interface{}) goqu.Expression {
	return table.Col(key).Eq(value)
}

func Neq(table exp.AliasedExpression, key string, value interface{}) goqu.Expression {
	return table.Col(key).Neq(value)
}

func Like(table exp.AliasedExpression, key string, value interface{}) goqu.Expression {
	return table.Col(key).Like(value)
}

func ILike(table exp.AliasedExpression, key string, value interface{}) goqu.Expression {
	return table.Col(key).ILike(value)
}

func In(table exp.AliasedExpression, key string, value interface{}) goqu.Expression {
	return table.Col(key).In(value)
}

func NotIn(table exp.AliasedExpression, key string, value interface{}) goqu.Expression {
	return table.Col(key).NotIn(value)
}

func IsNull(table exp.AliasedExpression, key string, value interface{}) goqu.Expression {
	if cast.ToBool(value) {
		return table.Col(key).IsNull()
	} else {
		return table.Col(key).IsNotNull()
	}
}
