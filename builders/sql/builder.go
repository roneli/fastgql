package sql

import (
	"fastgql/builders"
	"fmt"
	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/spf13/cast"
	"github.com/vektah/gqlparser/v2/ast"
	"math/rand"
	"unsafe"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyz"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

func GenerateTableName(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, rand.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = rand.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return *(*string)(unsafe.Pointer(&b))
}

type column struct {
	name string
	tableName string
	alias string
}

type QueryBuilder struct {
	builder *goqu.SelectDataset
	name string
	table exp.AliasedExpression
	columns []column
}

func NewBuilder(tableName string) QueryBuilder {
	genName := GenerateTableName(6)
	table := goqu.T(tableName).As(genName)
	return QueryBuilder{goqu.From(table), genName, table, nil}
}

func (qb QueryBuilder) Query() (string, []interface{}, error) {
	query, args, err := qb.builder.Select(buildJsonObjectExp(qb.columns, "")).ToSQL()
	fmt.Println(query)
	return query, args ,err
}

func (qb QueryBuilder) Builder() *goqu.SelectDataset {
	return qb.builder.Clone().(*goqu.SelectDataset)
}

func (qb *QueryBuilder) OnSingleField(f *ast.Field, variables map[string]interface{}) error {
	// add simple fields as columns
	qb.columns = append(qb.columns, column{name: f.Name, alias: f.Alias, tableName: qb.name})
	return nil
}

func (qb *QueryBuilder) OnMultiField(f *ast.Field, variables map[string]interface{}) error {
	d := f.Definition.Directives.ForName("sqlRelation")
	if d != nil {
		rel := parseRelationDirective(d)
		err := qb.buildRelation(rel, f, variables)
		if err != nil {
			return err
		}
	}
	return nil
}


func (qb *QueryBuilder) OnCollectStart(f *ast.Field, variables map[string]interface{}) error {

	limitArg := f.Arguments.ForName("limit")
	if limitArg != nil {
		limit, err := limitArg.Value.Value(variables)
		if err != nil {
			return err
		}
		qb.builder = qb.builder.Limit(cast.ToUint(limit))
	}

	offsetArg := f.Arguments.ForName("offset")
	if offsetArg != nil {
		offset, err := offsetArg.Value.Value(variables)
		if err != nil {
			return err
		}
		qb.builder = qb.builder.Offset(cast.ToUint(offset))
	}

	filterArg := f.Arguments.ForName("filter")
	if offsetArg != nil {
		filter, err := filterArg.Value.Value(variables)
		if err != nil {
			return err
		}
		filterMap, ok := filter.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid filter type")
		}
		if err := qb.buildCondition(filterMap); err != nil {
			return err
		}
	}

	return nil
}

func (qb *QueryBuilder) buildCondition(filter map[string]interface{}) error {
	fmt.Print(filter)
	return nil
}

func (qb *QueryBuilder) buildRelation(rel relation, f *ast.Field, variables map[string]interface{}) error {

	builder := NewBuilder(rel.relationTableName)
	err := builders.CollectFields(&builder, f, variables)
	if err != nil {
		return err
	}
	switch rel.relType {
	case OneToOne:
		qb.builder = qb.builder.LeftJoin(
			goqu.Lateral(builder.Builder().Select(buildJsonObjectExp(
				builder.columns, f.Name)).As(builder.name)),
			buildJoinCondition(qb.name, rel.baseTableKeys, builder.name, rel.relationTableKeys),
			)
		qb.columns = append(qb.columns, column{name: f.Name, alias: "", tableName: builder.name})
	case OneToMany:
		qb.builder = qb.builder.LeftJoin(
			goqu.Lateral(builder.Builder().Select(buildJsonAgg(builder.columns, f.Name)).As(builder.name)),
			buildJoinCondition(qb.name, rel.baseTableKeys, builder.name, rel.relationTableKeys),
			)
		qb.columns = append(qb.columns, column{name: f.Name, alias: "", tableName: builder.name})
	}
	return err
}

func buildJsonAgg(columns []column, alias string) exp.Expression {
	return goqu.COALESCE(goqu.Func("jsonb_agg", buildJsonObjectExp(columns, "")), goqu.L("[]")).As(alias)
}

func buildJsonObjectExp(columns []column, alias string) exp.Expression {
	var args []interface{}
	for _, c := range columns {
		args = append(args, goqu.I(c.name), goqu.I(fmt.Sprintf("%s.%s", c.tableName, c.name)))
	}
	buildJsonObj := goqu.Func("jsonb_build_object", args...)
	if alias != "" {
		return buildJsonObj.As(alias)
	}
	return buildJsonObj
}

func buildJoinCondition(leftTableName string, leftKeys []string, rightTableName string, rightKeys []string) exp.JoinCondition {
	var keys = make([]exp.Expression, len(leftKeys))
	for i := range leftKeys {
		keys[i] = goqu.Ex{fmt.Sprintf("%s.%s", leftTableName, leftKeys[i]): fmt.Sprintf("%s.%s", rightTableName, rightKeys[i])}
	}
	return goqu.On(keys...)
}