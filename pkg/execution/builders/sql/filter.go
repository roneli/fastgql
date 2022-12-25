package sql

import (
	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/roneli/fastgql/pkg/execution/builders"
	"github.com/spf13/cast"
)

var defaultOperators = map[string]builders.Operator{
	"eq":     Eq,
	"neq":    Neq,
	"like":   Like,
	"ilike":  ILike,
	"notIn":  NotIn,
	"in":     In,
	"isNull": IsNull,
	"gt":     Gt,
	"gte":    Gte,
	"lte":    Lte,
	"lt":     Lt,
}

func Eq(table exp.AliasedExpression, key string, value interface{}) goqu.Expression {
	return table.Col(key).Eq(value)
}

func Neq(table exp.AliasedExpression, key string, value interface{}) goqu.Expression {
	return table.Col(key).Neq(value)
}

func Gt(table exp.AliasedExpression, key string, value interface{}) goqu.Expression {
	return table.Col(key).Gt(value)
}

func Gte(table exp.AliasedExpression, key string, value interface{}) goqu.Expression {
	return table.Col(key).Gte(value)
}

func Lte(table exp.AliasedExpression, key string, value interface{}) goqu.Expression {
	return table.Col(key).Lte(value)
}

func Lt(table exp.AliasedExpression, key string, value interface{}) goqu.Expression {
	return table.Col(key).Lt(value)
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
