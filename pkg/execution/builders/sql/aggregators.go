package sql

import (
	"fmt"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/iancoleman/strcase"

	"github.com/roneli/fastgql/pkg/execution/builders"
)

var defaultAggregatorOperators = map[string]builders.AggregatorOperator{
	"max": MaxAggregator,
	"min": MinAggregator,
	"avg": AvgAggregator,
	"sum": SumAggregator,
}

func SumAggregator(table exp.AliasedExpression, fields []builders.Field) (goqu.Expression, error) {
	sumFields := make([]interface{}, 0, len(fields)*2)
	for _, f := range fields {
		sumFields = append(sumFields, goqu.L(fmt.Sprintf("'%s'", f.Name)), goqu.SUM(table.Col(strcase.ToSnake(f.Name))))
	}
	return goqu.Func("json_build_object", sumFields...), nil
}

func AvgAggregator(table exp.AliasedExpression, fields []builders.Field) (goqu.Expression, error) {
	avgFields := make([]interface{}, 0, len(fields)*2)
	for _, f := range fields {
		avgFields = append(avgFields, goqu.L(fmt.Sprintf("'%s'", f.Name)), goqu.AVG(table.Col(strcase.ToSnake(f.Name))))
	}
	return goqu.Func("json_build_object", avgFields...), nil
}

func MaxAggregator(table exp.AliasedExpression, fields []builders.Field) (goqu.Expression, error) {
	maxFields := make([]interface{}, 0, len(fields)*2)
	for _, f := range fields {
		maxFields = append(maxFields, goqu.L(fmt.Sprintf("'%s'", f.Name)), goqu.MAX(table.Col(strcase.ToSnake(f.Name))))
	}
	return goqu.Func("json_build_object", maxFields...), nil
}

func MinAggregator(table exp.AliasedExpression, fields []builders.Field) (goqu.Expression, error) {
	minFields := make([]interface{}, 0, len(fields)*2)
	for _, f := range fields {
		minFields = append(minFields, goqu.L(fmt.Sprintf("'%s'", f.Name)), goqu.MIN(table.Col(strcase.ToSnake(f.Name))))
	}
	return goqu.Func("json_build_object", minFields...), nil
}
