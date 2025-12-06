package sql

import (
	"fmt"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/roneli/fastgql/pkg/execution/builders"
	"github.com/spf13/cast"
)

var defaultOperators = map[string]builders.Operator{
	"eq":     opEq,
	"neq":    opNeq,
	"like":   opLike,
	"ilike":  opILike,
	"notIn":  opNotIn,
	"in":     opIn,
	"isNull": opIsNull,
	"gt":     opGt,
	"gte":    opGte,
	"lte":    opLte,
	"lt":     opLt,
	"prefix": opPrefix,
	"suffix": opSuffix,
}

func opEq(table exp.AliasedExpression, key string, value interface{}) goqu.Expression {
	return table.Col(key).Eq(value)
}

func opNeq(table exp.AliasedExpression, key string, value interface{}) goqu.Expression {
	return table.Col(key).Neq(value)
}

func opGt(table exp.AliasedExpression, key string, value interface{}) goqu.Expression {
	return table.Col(key).Gt(value)
}

func opGte(table exp.AliasedExpression, key string, value interface{}) goqu.Expression {
	return table.Col(key).Gte(value)
}

func opLte(table exp.AliasedExpression, key string, value interface{}) goqu.Expression {
	return table.Col(key).Lte(value)
}

func opLt(table exp.AliasedExpression, key string, value interface{}) goqu.Expression {
	return table.Col(key).Lt(value)
}

func opLike(table exp.AliasedExpression, key string, value interface{}) goqu.Expression {
	return table.Col(key).Like(value)
}

func opILike(table exp.AliasedExpression, key string, value interface{}) goqu.Expression {
	return table.Col(key).ILike(value)
}

func opIn(table exp.AliasedExpression, key string, value interface{}) goqu.Expression {
	return table.Col(key).In(value)
}

func opNotIn(table exp.AliasedExpression, key string, value interface{}) goqu.Expression {
	return table.Col(key).NotIn(value)
}

func opIsNull(table exp.AliasedExpression, key string, value interface{}) goqu.Expression {
	if cast.ToBool(value) {
		return table.Col(key).IsNull()
	} else {
		return table.Col(key).IsNotNull()
	}
}

func opPrefix(table exp.AliasedExpression, key string, value interface{}) goqu.Expression {
	return table.Col(key).Like(fmt.Sprintf("%s%%", value))
}

func opSuffix(table exp.AliasedExpression, key string, value interface{}) goqu.Expression {
	return table.Col(key).Like(fmt.Sprintf("%%%s", value))
}
