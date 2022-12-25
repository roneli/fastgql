package sql

import (
	"fmt"

	builders2 "github.com/roneli/fastgql/pkg/execution/builders"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/iancoleman/strcase"
)

var defaultAggregatorOperators = map[string]builders2.AggregatorOperator{
	"max": MaxAggregator,
	"min": MinAggregator,
}

func MaxAggregator(table exp.AliasedExpression, fields []builders2.Field) (goqu.Expression, error) {
	maxFields := make([]interface{}, 0, len(fields)*2)
	for _, f := range fields {
		maxFields = append(maxFields, goqu.L(fmt.Sprintf("'%s'", f.Name)), goqu.MAX(table.Col(strcase.ToSnake(f.Name))))
	}
	return goqu.Func("json_build_object", maxFields...), nil
}

func MinAggregator(table exp.AliasedExpression, fields []builders2.Field) (goqu.Expression, error) {
	minFields := make([]interface{}, 0, len(fields)*2)
	for _, f := range fields {
		minFields = append(minFields, goqu.L(fmt.Sprintf("'%s'", f.Name)), goqu.MIN(table.Col(strcase.ToSnake(f.Name))))
	}
	return goqu.Func("json_build_object", minFields...), nil
}
