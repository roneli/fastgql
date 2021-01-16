package sql

import (
	"errors"
	"fastgql/builders"
	"fastgql/schema"
	"fmt"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/iancoleman/strcase"
	"github.com/vektah/gqlparser/v2/ast"
)

func newExpressionBuilder(b *Builder, logicalType schema.LogicalOperator) *expressionsBuilder{
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


type expressionsBuilder struct {
	exp.ExpressionList
	builder *Builder
}

func (e *expressionsBuilder) Operation(name, key string, value interface{}) error {

	op, ok := e.builder.operators[key]
	if !ok {
		return fmt.Errorf("key operator %s not supported", key)
	}
	e.ExpressionList = e.Append(op(e.builder.table, strcase.ToSnake(name), value))
	return nil
}

func (e *expressionsBuilder) Filter(f *ast.Field, key string, values map[string]interface{}) error {
	return nil
}

func (e *expressionsBuilder) Logical(f *ast.Field, logicalExp schema.LogicalOperator, values []interface{}) error {

	switch logicalExp {
	case schema.LogicalOperatorOR, schema.LogicalOperatorAND:
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
	case schema.LogicalOperatorNot:
		return errors.New("not implemented")
	}
	return nil
}