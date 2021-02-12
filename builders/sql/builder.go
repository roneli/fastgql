package sql

import (
	"fmt"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/roneli/fastgql/builders"
	"github.com/roneli/fastgql/schema"

	// import the dialect
	_ "github.com/doug-martin/goqu/v9/dialect/postgres"
	"github.com/iancoleman/strcase"
	"github.com/spf13/cast"
	"github.com/vektah/gqlparser/v2/ast"
)

type column struct {
	name      string
	tableName string
	alias     string
}

type Builder struct {
	// base builders config, includes logger, schema etc'
	config *builders.Config
	// operators supported for filtering
	operators map[string]Operator
	// actual goqu query sqlbuilder
	builder *goqu.SelectDataset
	// Name of table, this will be the generated name
	name string
	// table expression used by the goqu builder
	table exp.AliasedExpression
	// columns collected by builders.BuildFields these columns are used to buildJsonObject or buildJsonAgg
	columns []column
}

var defaultOperators = map[string]Operator{
	"eq":     Eq,
	"neq":    Neq,
	"like":   Like,
	"ilike":  ILike,
	"notIn":  NotIn,
	"in":     In,
	"isNull": IsNull,
	"gt": Gt,
	"gte": Gte,
	"lte": Lte,
	"lt": Lt,
}

// NewBuilder is the entry point for creating builders
func NewBuilder(cfg *builders.Config, field *ast.Field) (Builder, error) {
	genName := GenerateTableName(6)
	table := goqu.T(getTableName(cfg.Schema, field.Definition)).As(genName)
	return Builder{cfg, defaultOperators, goqu.From(table), genName, table, nil}, nil
}

func (b Builder) Config() *builders.Config {
	return b.config
}

func (b Builder) Query() (string, []interface{}, error) {
	b.builder = b.Select()
	query, args, err := b.builder.WithDialect("postgres").Prepared(true).ToSQL()
	fmt.Println(query, args)
	return query, args, err
}

func (b *Builder) Select() *goqu.SelectDataset {
	for i, c := range b.columns {
		if i == 0 {
			b.builder = b.builder.Select(goqu.T(c.tableName).Col(c.name).As(c.name))
		} else {
			b.builder = b.builder.SelectAppend(goqu.T(c.tableName).Col(c.name).As(c.name))
		}
	}
	return b.builder
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
		err := b.buildRelationQuery(rel, f, variables)
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
			b.builder = b.builder.OrderAppend(goqu.C(strcase.ToSnake(o.Key)).Asc().NullsLast())
		case builders.OrderingTypesAscNull:
			b.builder = b.builder.OrderAppend(goqu.C(strcase.ToSnake(o.Key)).Asc().NullsFirst())
		case builders.OrderingTypesDesc:
			b.builder = b.builder.OrderAppend(goqu.C(strcase.ToSnake(o.Key)).Desc().NullsLast())
		case builders.OrderingTypesDescNull:
			b.builder = b.builder.OrderAppend(goqu.C(strcase.ToSnake(o.Key)).Desc().NullsFirst())
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

func (b *Builder) Filter(f *ast.FieldDefinition, key string, value map[string]interface{}) error {
	fieldDef := b.Config().Schema.Types[f.Type.Name()]
	filterFieldDef := fieldDef.Fields.ForName(key)
	// Create a builder
	d := filterFieldDef.Directives.ForName("sqlRelation")
	if d == nil {
		return fmt.Errorf("missing directive sqlRelation")
	}
	fb, err := buildFilterInput(b, parseRelationDirective(d))
	if err != nil {
		return err
	}

	if err := builders.BuildFilter(&fb, filterFieldDef, value); err != nil {
		return err
	}
	b.builder = b.builder.Where(goqu.Func("exists", fb.builder))
	return nil
}

func (b *Builder) Logical(f *ast.FieldDefinition, logicalExp schema.LogicalOperator, values []interface{}) error {

	expList := newExpressionBuilder(b, logicalExp)
	if err := expList.Logical(f, logicalExp, values); err != nil {
		return err
	}
	switch logicalExp {
	case schema.LogicalOperatorOR, schema.LogicalOperatorAND:
		b.builder = b.builder.Where(expList)
	case schema.LogicalOperatorNot:
		b.builder = b.builder.Where(goqu.Func("NOT", expList))
	}
	return nil
}

// ======================================== Internal Methods ============================================

// createInnerBuilder creates an internal builder for a relation or filter query. The internal builder uses the
// same configuration as defined form the parent builder.
func (b *Builder) createInnerBuilder(tableName string) (Builder, error) {

	tableAlias := GenerateTableName(6)
	table := goqu.T(tableName).As(tableAlias)
	return Builder{
		config:    b.config,
		operators: b.operators,
		builder:   goqu.Dialect("postgres").From(table),
		name:      tableAlias,
		table:     table,
		columns:   nil,
	}, nil
}

func (b *Builder) buildRelationQuery(rel relation, f *ast.Field, variables map[string]interface{}) error {
	relBuilder, _ := b.createInnerBuilder(rel.referenceTable)
	if err := builders.BuildFields(&relBuilder, f, variables); err != nil {
		return err
	}
	if err := builders.BuildArguments(&relBuilder, f, variables); err != nil {
		return err
	}

	switch rel.relType {
	case OneToOne:
		b.builder = b.builder.LeftJoin(
			goqu.Lateral(relBuilder.builder.Select(buildJsonObject(
				relBuilder.columns, f.Name)).As(relBuilder.name).
				Where(buildCrossCondition(b.name, rel.fields, relBuilder.name, rel.references))),
			goqu.On(goqu.L("true")),
		)
		b.columns = append(b.columns, column{name: f.Name, alias: "", tableName: relBuilder.name})
	case OneToMany:
		b.builder = b.builder.LeftJoin(
			goqu.Lateral(relBuilder.builder.Select(buildJsonAgg(relBuilder.columns, f.Name)).As(relBuilder.name).
				Where(buildCrossCondition(b.name, rel.fields, relBuilder.name, rel.references))),
			goqu.On(goqu.L("true")),
		)
		b.columns = append(b.columns, column{name: f.Name, alias: "", tableName: relBuilder.name})
	case ManyToMany:

		relBuilder, _ := b.createInnerBuilder(rel.referenceTable)
		if err := builders.BuildFields(&relBuilder, f, variables); err != nil {
			return err
		}
		// Join the referenced table to the m2mQuery
		m2mBuilder, _ := relBuilder.createInnerBuilder(rel.manyToManyTable)
		// pass relBuilder columns to m2mBuilder since we are left joining relBuilder selection
		m2mBuilder.columns = relBuilder.columns
		// Build arguments on m2m table after it was joined with relation table
		if err := builders.BuildArguments(&m2mBuilder, f, variables); err != nil {
			return err
		}
		// Join m2mBuilder with the relBuilder
		m2mBuilder.builder = m2mBuilder.builder.LeftJoin(
			goqu.Lateral(relBuilder.Select().Where(buildCrossCondition(relBuilder.name, rel.references, m2mBuilder.name, rel.manyToManyReferences))).As(relBuilder.name),
			goqu.On(goqu.L("true")))
		// Add cross condition from parent builder (current Builder instance)
		m2mBuilder.builder = m2mBuilder.builder.Where(buildCrossCondition(b.name, rel.fields, m2mBuilder.name, rel.manyToManyFields)).As(relBuilder.name)

		// Finally aggregate relation query and join the m2m table with the main query
		aggTableName := GenerateTableName(6)
		aggQuery := goqu.From(m2mBuilder.Select()).As(aggTableName).Select(
			goqu.COALESCE(goqu.Func("jsonb_agg", buildJsonObject(relBuilder.columns, "")), goqu.L("'[]'::jsonb")).As(f.Name)).As(aggTableName)
		b.builder = b.builder.CrossJoin(goqu.Lateral(aggQuery))
		b.columns = append(b.columns, column{name: f.Name, alias: "", tableName: aggTableName})
	}
	return nil
}

// ====================================================================================================================
//      											Helper Functions
// ====================================================================================================================

// buildJsonAgg collects all the column values, including nulls, into a JSON array. Values are converted to JSON as per to_json or to_jsonb.
// See https://www.postgresql.org/docs/current/functions-aggregate.html for more information on this function.
func buildJsonAgg(columns []column, alias string) exp.Expression {
	return goqu.COALESCE(goqu.Func("jsonb_agg", buildJsonObject(columns, "")), goqu.L("'[]'::jsonb")).As(alias)
}

func buildJsonObject(columns []column, alias string) exp.Expression {
	args := make([]interface{}, len(columns)*2)
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
	return goqu.And(buildJoinCondition(leftTableName, leftKeys, rightTableName, rightKeys)...)
}

func buildJoinCondition(leftTableName string, leftKeys []string, rightTableName string, rightKeys []string) []exp.Expression {
	var keys = make([]exp.Expression, len(leftKeys))
	for i := range leftKeys {
		keys[i] = goqu.L(fmt.Sprintf("%s.%s = %s.%s", leftTableName, leftKeys[i], rightTableName, rightKeys[i]))
	}
	return keys
}

func buildFilterInput(parentBuilder *Builder, rel relation) (Builder, error) {
	fq, _ := parentBuilder.createInnerBuilder(rel.referenceTable)
	fq.builder = fq.builder.Select(goqu.L("1")).From(fq.table)
	switch rel.relType {
	case ManyToMany:
		m2mTableName := GenerateTableName(6)
		jExps := buildJoinCondition(parentBuilder.name, rel.fields, m2mTableName, rel.manyToManyFields)
		jExps = append(jExps, buildJoinCondition(m2mTableName, rel.manyToManyReferences, fq.name, rel.references)...)
		fq.builder = fq.builder.InnerJoin(goqu.T(rel.manyToManyTable).As(m2mTableName), goqu.On(jExps...))
	case OneToOne:
		relationTableName := GenerateTableName(6)
		jExps := buildJoinCondition(parentBuilder.name, rel.fields, fq.name, rel.references)
		jExps = append(jExps, buildJoinCondition(parentBuilder.name, rel.fields, relationTableName, rel.fields)...)
		fq.builder = fq.builder.InnerJoin(goqu.T(rel.baseTable).As(relationTableName), goqu.On(jExps...))
	case OneToMany:
		fq.builder = fq.builder.InnerJoin(goqu.T(rel.baseTable).As(GenerateTableName(6)), goqu.On(buildJoinCondition(parentBuilder.name, rel.fields, fq.name, rel.references)...))
	default:
		panic("unknown relation type")
	}
	return fq, nil
}
