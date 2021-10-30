package sql

import (
	"fmt"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
)

type column struct {
	table string
	name  string
	alias string
}

func (c column) Expression() exp.AliasedExpression {
	return goqu.T(c.table).Col(c.name).As(c.name)
}

type queryHelper struct {
	*goqu.SelectDataset
	table   exp.AliasedExpression
	alias   string
	selects []column
}

type tableHelper struct {
	table exp.AliasedExpression
	alias string
}

func (q queryHelper) Table() tableHelper {
	return tableHelper{
		table: q.table,
		alias: q.alias,
	}
}

func (q queryHelper) TableName() string {
	return q.table.Aliased().(exp.IdentifierExpression).GetTable()
}

func (q queryHelper) SelectRow() *goqu.SelectDataset {
	for i, c := range q.selects {
		if i == 0 {
			q.SelectDataset = q.SelectDataset.Select(c.Expression())
		} else {
			q.SelectDataset = q.SelectDataset.SelectAppend(c.Expression())
		}
	}
	return q.SelectDataset.WithDialect("postgres").Prepared(true)
}

func (q queryHelper) SelectJson(alias string) *goqu.SelectDataset {
	buildJsonObj := q.buildJsonObject()
	if alias != "" {
		return q.Select(buildJsonObj.As(alias))
	}
	return q.Select(buildJsonObj).WithDialect("postgres").Prepared(true)
}

func (q queryHelper) SelectJsonAgg(alias string) *goqu.SelectDataset {
	return q.Select(q.buildJsonAgg(alias).As(alias)).As(alias).WithDialect("postgres").Prepared(true)
}

func (q queryHelper) SelectOne() *goqu.SelectDataset {
	return q.Select(goqu.L("1")).WithDialect("postgres").Prepared(true)
}

func (q queryHelper) buildJsonObject() exp.SQLFunctionExpression {
	args := make([]interface{}, len(q.selects)*2)
	for i, c := range q.selects {
		args[i*2] = goqu.L(fmt.Sprintf("'%s'", c.name))
		args[i*2+1] = goqu.I(fmt.Sprintf("%s.%s", c.table, c.name))
	}
	return goqu.Func("jsonb_build_object", args...)
}

func (q queryHelper) buildJsonAgg(alias string) exp.SQLFunctionExpression {
	return goqu.COALESCE(goqu.Func("jsonb_agg", q.buildJsonObject()), goqu.L("'[]'::jsonb"))
}

func buildCrossCondition(leftTableName string, leftKeys []string, rightTableName string, rightKeys []string) exp.ExpressionList {
	return goqu.And(buildJoinCondition(leftTableName, leftKeys, rightTableName, rightKeys)...)
}

func buildJoinCondition(leftTableName string, leftKeys []string, rightTableName string, rightKeys []string) []exp.Expression {
	var keys = make([]exp.Expression, len(leftKeys))
	for i := range leftKeys {
		keys[i] = goqu.L(fmt.Sprintf("%s.%s = %s.%s", leftTableName, leftKeys[i], rightTableName, rightKeys[i]))
	}
	return keys
}
