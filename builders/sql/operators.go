package sql

import (
	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
)

type Operator func(table exp.AliasedExpression, key string, value interface{}) goqu.Expression


func Eq(table exp.AliasedExpression, key string, value interface{}) goqu.Expression {
	return table.Col(key).Eq(value)
}

func Neq(table exp.AliasedExpression, key string, value interface{}) goqu.Expression {
	return table.Col(key).Neq(value)
}