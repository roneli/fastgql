package sql

import (
	"errors"
	"fastgql/builders"
	"fastgql/schema"
	"fmt"
	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	// import the dialect
	_ "github.com/doug-martin/goqu/v9/dialect/postgres"
	"github.com/iancoleman/strcase"
	"github.com/spf13/cast"
	"github.com/vektah/gqlparser/v2/ast"
)



type column struct {
	name string
	tableName string
	alias string
}

type Builder struct {
	operators map[string]Operator
	builder *goqu.SelectDataset
	name string
	table exp.AliasedExpression
	columns []column
}

var defaultOperators = map[string]Operator{
	"eq": Eq,
	"neq": Neq,
}

// NewBuilder is the entry point for creating builders
func NewBuilder(tableName string) (Builder, error) {
	genName := GenerateTableName(6)
	table := goqu.T(tableName).As(genName)
	return Builder{defaultOperators, goqu.From(table), genName, table, nil}, nil
}

func (b Builder) Query() (string, []interface{}, error) {


	for i, c := range b.columns {
		if i == 0 {
			b.builder = b.builder.Select(goqu.T(c.tableName).Col(c.name).As(c.name))
		} else {
			b.builder = b.builder.SelectAppend(goqu.T(c.tableName).Col(c.name).As(c.name))
		}
	}
	query, args, err := b.builder.WithDialect("postgres").Prepared(true).ToSQL()
	fmt.Println(query, args)
	return query, args ,err
}


func (b Builder) Type() string {
	return "SQL"
}

func (b Builder) Config() *builders.Config{
	return nil
}


func (b *Builder) OnSingleField(f *ast.Field, variables map[string]interface{}) error {
	// add simple fields as columns
	b.columns = append(b.columns, column{name: f.Name, alias: f.Alias, tableName: b.name})
	return nil
}

func (b *Builder) OnSelectionField(f *ast.Field, variables map[string]interface{}) error {
	d := f.Definition.Directives.ForName("sqlRelation")
	if d != nil {
		rel := parseRelationDirective(d)
		err := b.buildRelation(rel, f, variables)
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *Builder) Limit(limit uint) error {
	b.builder = b.builder.Limit(cast.ToUint(limit))
	return nil
}

func (b *Builder) Offset(offset uint) error {
	b.builder = b.builder.Offset(offset)
	return nil
}

func (b *Builder) OrderBy(orderFields []builders.OrderField) error {
	for _, o := range orderFields {
		switch o.Type {
		case builders.OrderingTypesAsc:
			b.builder = b.builder.OrderAppend(b.table.Col(strcase.ToSnake(o.Key)).Asc().NullsLast())
		case builders.OrderingTypesAscNull:
			b.builder = b.builder.OrderAppend(b.table.Col(strcase.ToSnake(o.Key)).Asc().NullsFirst())
		case builders.OrderingTypesDesc:
			b.builder = b.builder.OrderAppend(b.table.Col(strcase.ToSnake(o.Key)).Desc().NullsLast())
		case builders.OrderingTypesDescNull:
			b.builder = b.builder.OrderAppend(b.table.Col(strcase.ToSnake(o.Key)).Desc().NullsFirst())
		}
	}
	return nil
}

func (b *Builder) Operation(name, key string, value interface{}) error {
	op, ok := b.operators[key]
	if !ok {
		return fmt.Errorf("key operator %s not supported", key)
	}
	b.builder = b.builder.Where(op(b.table, strcase.ToSnake(name), value))
	return nil
}

func (b *Builder) Filter(f *ast.Field, key string, value map[string]interface{}) error {
	return nil
}

func (b *Builder) Logical(f *ast.Field, logicalExp schema.LogicalOperator, values []interface{}) error {
	switch logicalExp {
	case schema.LogicalOperatorOR, schema.LogicalOperatorAND:
		expList := newExpressionBuilder(b, logicalExp)
		if err := expList.Logical(f, logicalExp, values); err != nil {
			return err
		}
		b.builder = b.builder.Where(expList)
	case schema.LogicalOperatorNot:
		return errors.New("not implemented")
	}
	return nil
}

// ======================================== Internal Methods ============================================

func (b *Builder) buildRelation(rel relation, f *ast.Field, variables map[string]interface{}) error {

	builder, _ := NewBuilder(rel.referenceTable)
	err := builders.BuildFields(&builder, f, variables)
	if err != nil {
		return err
	}
	switch rel.relType {
	case OneToOne:
		b.builder = b.builder.LeftJoin(
			goqu.Lateral(builder.builder.Select(buildJsonObjectExp(
				builder.columns, f.Name)).As(builder.name).
				Where(buildCrossCondition(b.name, rel.fields, builder.name, rel.references))),
				goqu.On(goqu.L("true")),
			)
		b.columns = append(b.columns, column{name: f.Name, alias: "", tableName: builder.name})
	case OneToMany:
		b.builder = b.builder.LeftJoin(
			goqu.Lateral(builder.builder.Select(buildJsonAgg(builder.columns, f.Name)).As(builder.name).
			Where(buildCrossCondition(b.name, rel.fields, builder.name, rel.references))),
			goqu.On(goqu.L("true")),
			)
		b.columns = append(b.columns, column{name: f.Name, alias: "", tableName: builder.name})
	case ManyToMany:
		m2mTableName := GenerateTableName(6)
		m2mQuery := goqu.From(goqu.T(rel.manyToManyTable).As(m2mTableName)).Select(
			goqu.COALESCE(goqu.Func("jsonb_agg", builder.table.Col(f.Name)), goqu.L("'[]'::jsonb")).As(f.Name)).As(m2mTableName)
		// Join the referenced table to the m2mQuery
		m2mQuery = m2mQuery.LeftJoin(goqu.Lateral(builder.builder.Select(buildJsonObjectExp(builder.columns, f.Name)).
			As(builder.name).Where(buildCrossCondition(m2mTableName, rel.manyToManyReferences, builder.name, rel.references))),
			goqu.On(goqu.L("true")),
			)
		// Finally join the m2m table with the main query
		b.builder = b.builder.LeftJoin(goqu.Lateral(m2mQuery.Where(buildCrossCondition(b.name, rel.fields, m2mTableName, rel.manyToManyFields))), goqu.On(goqu.L("true")))
		b.columns = append(b.columns, column{name: f.Name, alias: "", tableName: m2mTableName})
	}
	return err
}

func buildJsonAgg(columns []column, alias string) exp.Expression {
	return goqu.COALESCE(goqu.Func("jsonb_agg", buildJsonObjectExp(columns, "")), goqu.L("'[]'::jsonb")).As(alias)
}

func buildJsonObjectExp(columns []column, alias string) exp.Expression {
	var args []interface{}
	for _, c := range columns {
		args = append(args, goqu.L(fmt.Sprintf("'%s'", c.name)), goqu.I(fmt.Sprintf("%s.%s", c.tableName, c.name)))
	}
	buildJsonObj := goqu.Func("jsonb_build_object", args...)
	if alias != "" {
		return buildJsonObj.As(alias)
	}
	return buildJsonObj
}

func buildCrossCondition(leftTableName string, leftKeys []string, rightTableName string, rightKeys []string) exp.ExpressionList {
	var keys = make([]exp.Expression, len(leftKeys))
	for i := range leftKeys {
		keys[i] = goqu.L(fmt.Sprintf("%s.%s = %s.%s", leftTableName, leftKeys[i], rightTableName, rightKeys[i]))
	}
	return goqu.And(keys...)
}


func buildJoinCondition(leftTableName string, leftKeys []string, rightTableName string, rightKeys []string) exp.JoinCondition {
	var keys = make([]exp.Expression, len(leftKeys))
	for i := range leftKeys {
		keys[i] = goqu.L(fmt.Sprintf("%s.%s = %s.%s", leftTableName, leftKeys[i], rightTableName, rightKeys[i]))
	}
	return goqu.On(keys...)
}