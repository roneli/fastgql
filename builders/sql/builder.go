package sql

import (
	"context"
	"fmt"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/iancoleman/strcase"
	"github.com/pkg/errors"
	"github.com/roneli/fastgql/builders"
	"github.com/roneli/fastgql/log"
	"github.com/roneli/fastgql/schema"
	"github.com/spf13/cast"
	"github.com/vektah/gqlparser/v2/ast"

	"strings"
	// import the dialect
	_ "github.com/doug-martin/goqu/v9/dialect/postgres"
)

type builder struct {
	Schema    *ast.Schema
	Logger    log.Logger
	Operators map[string]Operator
}

func NewBuilder(config *builders.Config) builder {
	var l log.Logger = log.NullLogger{}
	if config.Logger != nil {
		l = config.Logger
	}
	return builder{Schema: config.Schema, Logger: l, Operators: defaultOperators}
}

func (b builder) Query(ctx context.Context) (string, []interface{}, error) {
	field := builders.CollectFields(ctx)
	query, err := b.buildQuery(getTableName(b.Schema, field.Definition), field)
	if err != nil {
		return "", nil, err
	}
	return query.SelectRow().ToSQL()
}

func (b builder) Aggregate(ctx context.Context) (string, []interface{}, error) {
	field := builders.CollectFields(ctx)
	query, err := b.buildAggregate(getAggregateTableName(b.Schema, field.Field), field)
	if err != nil {
		return "", nil, err
	}
	return query.ToSQL()
}

func (b builder) buildQuery(tableName string, field builders.Field) (*queryHelper, error) {

	b.Logger.Debug("building query", map[string]interface{}{"table": tableName})
	tableAlias := builders.GenerateTableName(6)
	table := goqu.T(tableName).As(tableAlias)
	query := queryHelper{goqu.From(table), table, tableAlias, nil}

	// Add field columns
	for _, childField := range field.Selections {
		switch childField.FieldType {
		case builders.TypeScalar:
			b.Logger.Debug("adding field", map[string]interface{}{"table": tableName, "fieldName": childField.Name})
			query.selects = append(query.selects, column{table: query.alias, name: childField.Name, alias: ""})
		case builders.TypeRelation:
			b.Logger.Debug("adding relation field", map[string]interface{}{"table": tableName, "fieldName": childField.Name})
			if err := b.buildRelation(&query, childField); err != nil {
				return nil, fmt.Errorf("failed to build relation for %s", childField.Name)
			}
		case builders.TypeAggregate:
			//aggQuery, err := b.buildRelationAggregate(getTableName(b.Schema, childField.Definition), childField)
			//if err != nil {
			//	return nil, err
			//}
			//query.selects = append(query.selects, column{table: query.alias, name: childField.Name, alias: ""})

		default:
			b.Logger.Error("unknown field type", map[string]interface{}{"table": tableName, "fieldName": childField.Name, "fieldType": childField.FieldType})
			panic("unknown field type")
		}
	}
	b.buildPagination(&query, field)
	b.buildOrdering(&query, field)
	if err := b.buildFiltering(&query, field); err != nil {
		return nil, err
	}

	return &query, nil
}

func (b builder) buildAggregate(tableName string, field builders.Field) (*queryHelper, error) {
	b.Logger.Debug("building aggregate", map[string]interface{}{"table": tableName})
	tableAlias := builders.GenerateTableName(6)
	table := goqu.T(tableName).As(tableAlias)
	query := &queryHelper{goqu.From(table), table, tableAlias, nil}

	var aggColumns []interface{}
	for _, f := range field.Selections {
		if f.Name == "count" {
			aggColumns = append(aggColumns, "count", goqu.COUNT(goqu.L("1")))
		}
	}
	if err := b.buildFiltering(query, field); err != nil {
		return nil, err
	}

	query.SelectDataset = query.Select(goqu.Func("json_build_object", aggColumns...)).As(field.Name)
	return query, nil
}

func (b builder) buildOrdering(query *queryHelper, field builders.Field) {
	orderBy, ok := field.Arguments["orderBy"]
	if !ok {
		return
	}
	orderFields, _ := builders.CollectOrdering(orderBy)

	for _, o := range orderFields {
		b.Logger.Debug("adding ordering", map[string]interface{}{"table": query.TableName(), "field": o.Key, "orderType": o.Type})
		switch o.Type {
		case builders.OrderingTypesAsc:
			query.SelectDataset = query.OrderAppend(goqu.C(strcase.ToSnake(o.Key)).Asc().NullsLast())
		case builders.OrderingTypesAscNull:
			query.SelectDataset = query.OrderAppend(goqu.C(strcase.ToSnake(o.Key)).Asc().NullsFirst())
		case builders.OrderingTypesDesc:
			query.SelectDataset = query.OrderAppend(goqu.C(strcase.ToSnake(o.Key)).Desc().NullsLast())
		case builders.OrderingTypesDescNull:
			query.SelectDataset = query.OrderAppend(goqu.C(strcase.ToSnake(o.Key)).Desc().NullsFirst())
		}
	}
}

func (b builder) buildPagination(query *queryHelper, field builders.Field) {

	if limit, ok := field.Arguments["limit"]; ok {
		b.Logger.Debug("adding pagination limit", map[string]interface{}{"table": query.TableName(), "limit": limit})
		query.SelectDataset = query.Limit(cast.ToUint(limit))
	}
	if offset, ok := field.Arguments["offset"]; ok {
		b.Logger.Debug("adding pagination offset", map[string]interface{}{"table": query.TableName(), "offset": offset})
		query.SelectDataset = query.Offset(cast.ToUint(offset))
	}

}

func (b builder) buildFiltering(query *queryHelper, field builders.Field) error {
	filterArg, ok := field.Arguments["filter"]
	if !ok {
		return nil
	}
	filters, ok := filterArg.(map[string]interface{})
	if !ok {
		return fmt.Errorf("unexpected filter arg type")
	}
	filterExp, _ := b.buildFilterExp(query, field, filters)
	query.SelectDataset = query.Where(filterExp)
	return nil
}

func (b builder) buildFilterLogicalExp(query *queryHelper, field builders.Field, filtersList []interface{}, logicalType exp.ExpressionListType) (goqu.Expression, error) {

	expBuilder := exp.NewExpressionList(logicalType)
	for _, filterValue := range filtersList {
		kv, ok := filterValue.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("fatal value of bool exp not map")
		}
		filterExp, err := b.buildFilterExp(query, field, kv)
		if err != nil {
			return nil, err
		}
		expBuilder = expBuilder.Append(filterExp)
	}
	return expBuilder, nil
}

func (b builder) buildFilterExp(query *queryHelper, field builders.Field, filters map[string]interface{}) (goqu.Expression, error) {
	filterInputDef := b.Schema.Types[fmt.Sprintf("%sFilterInput", field.GetTypeName())]
	expBuilder := exp.NewExpressionList(exp.AndType)
	for k, v := range filters {
		keyType := filterInputDef.Fields.ForName(k).Type
		switch {
		case k == string(schema.LogicalOperatorAND) || k == string(schema.LogicalOperatorOR):
			vv, ok := v.([]interface{})
			if !ok {
				return nil, fmt.Errorf("fatal value of logical list exp not list")
			}
			logicalType := exp.AndType
			if k == string(schema.LogicalOperatorOR) {
				logicalType = exp.OrType
			}
			logicalExp, err := b.buildFilterLogicalExp(query, field, vv, logicalType)
			if err != nil {
				return nil, err
			}
			expBuilder = expBuilder.Append(logicalExp)
		case k == string(schema.LogicalOperatorNot):
			filterExp, err := b.buildFilterExp(query, field, filters)
			if err != nil {
				return nil, err
			}
			expBuilder = expBuilder.Append(goqu.Func("NOT", filterExp))
		case strings.HasSuffix(keyType.Name(), "FilterInput"):
			kv, ok := v.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("fatal value of bool exp not map")
			}
			fieldDef := b.Schema.Types[field.GetTypeName()]
			filterFieldDef := fieldDef.Fields.ForName(k)
			// Create a builder
			d := filterFieldDef.Directives.ForName("sqlRelation")
			if d == nil {
				return nil, fmt.Errorf("missing directive sqlRelation")
			}
			fq, err := b.buildFilterQuery(query, field, parseRelationDirective(d), kv)
			if err != nil {
				return nil, err
			}
			expBuilder = expBuilder.Append(goqu.Func("exists", fq.SelectOne()))
		default:
			opMap, ok := v.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("fatal value of key not map")
			}
			for op, value := range opMap {
				opExp, err := b.Operation(query.table, k, op, value)
				if err != nil {
					return nil, err
				}
				expBuilder = expBuilder.Append(opExp)
			}
		}
	}
	return expBuilder, nil
}

func (b builder) Operation(table exp.AliasedExpression, fieldName, operatorName string, value interface{}) (goqu.Expression, error) {
	opFunc, ok := b.Operators[operatorName]
	if !ok {
		return nil, fmt.Errorf("key operator %s not supported", operatorName)
	}
	return opFunc(table, strcase.ToSnake(fieldName), value), nil
}

func (b builder) buildRelation(parentQuery *queryHelper, rf builders.Field) error {

	relationQuery, err := b.buildQuery(rf.Field.Name, rf)
	if err != nil {
		return errors.Wrap(err, "failed building relation")
	}
	rel := parseRelationDirective(rf.Definition.Directives.ForName("sqlRelation"))
	switch rel.relType {
	case OneToOne:
		parentQuery.SelectDataset = parentQuery.SelectDataset.LeftJoin(goqu.Lateral(relationQuery.SelectJson(rf.Name).As(relationQuery.alias).
			Where(buildCrossCondition(parentQuery.alias, rel.fields, relationQuery.alias, rel.references))),
			goqu.On(goqu.L("true")),
		)
		parentQuery.selects = append(parentQuery.selects, column{name: rf.Name, alias: "", table: relationQuery.alias})
	case OneToMany:
		parentQuery.SelectDataset = parentQuery.SelectDataset.LeftJoin(
			goqu.Lateral(relationQuery.SelectJsonAgg(rf.Name).As(relationQuery.alias).
				Where(buildCrossCondition(parentQuery.alias, rel.fields, relationQuery.alias, rel.references))),
			goqu.On(goqu.L("true")),
		)
		parentQuery.selects = append(parentQuery.selects, column{name: rf.Name, alias: "", table: relationQuery.alias})
	case ManyToMany:
		m2mTableAlias := builders.GenerateTableName(6)
		m2mTable := goqu.T(rel.manyToManyTable).As(m2mTableAlias)
		m2mQuery := queryHelper{
			SelectDataset: goqu.From(m2mTable),
			table:         m2mTable,
			alias:         m2mTableAlias,
			selects:       relationQuery.selects,
		}
		// Join m2mBuilder with the relBuilder
		m2mQuery.SelectDataset = m2mQuery.LeftJoin(
			goqu.Lateral(relationQuery.SelectRow().Where(buildCrossCondition(relationQuery.alias, rel.references, m2mTableAlias, rel.manyToManyReferences))).As(relationQuery.alias),
			goqu.On(goqu.L("true")))

		// Add cross condition from parent builder (current builder instance)
		m2mQuery.SelectDataset = m2mQuery.Where(buildCrossCondition(parentQuery.alias, rel.fields, m2mTableAlias, rel.manyToManyFields)).As(relationQuery.alias)

		// Finally aggregate relation query and join the m2m table with the main query
		aggTableName := builders.GenerateTableName(6)
		aggQuery := goqu.From(m2mQuery.SelectRow()).As(aggTableName).Select(relationQuery.buildJsonAgg(rf.Name).As(rf.Name)).As(aggTableName)
		parentQuery.SelectDataset = parentQuery.CrossJoin(goqu.Lateral(aggQuery))
		parentQuery.selects = append(parentQuery.selects, column{name: rf.Name, alias: "", table: aggTableName})

	}
	return nil
}

func (b builder) buildFilterQuery(parentQuery *queryHelper, rf builders.Field, rel relation, filters map[string]interface{}) (*queryHelper, error) {

	tableAlias := builders.GenerateTableName(6)
	table := goqu.T(rel.referenceTable).As(tableAlias)
	fq := &queryHelper{goqu.From(table), table, tableAlias, nil}

	switch rel.relType {
	case ManyToMany:
		m2mTableName := builders.GenerateTableName(6)
		jExps := buildJoinCondition(parentQuery.alias, rel.fields, m2mTableName, rel.manyToManyFields)
		jExps = append(jExps, buildJoinCondition(m2mTableName, rel.manyToManyReferences, fq.alias, rel.references)...)
		fq.SelectDataset = fq.InnerJoin(goqu.T(rel.manyToManyTable).As(m2mTableName), goqu.On(jExps...))
	case OneToOne:
		relationTableName := builders.GenerateTableName(6)
		jExps := buildJoinCondition(parentQuery.alias, rel.fields, fq.alias, rel.references)
		jExps = append(jExps, buildJoinCondition(parentQuery.alias, rel.fields, relationTableName, rel.fields)...)
		fq.SelectDataset = fq.InnerJoin(goqu.T(rel.baseTable).As(relationTableName), goqu.On(jExps...))
	case OneToMany:
		fq.SelectDataset = fq.InnerJoin(goqu.T(rel.baseTable).As(builders.GenerateTableName(6)), goqu.On(buildJoinCondition(parentQuery.alias, rel.fields, fq.alias, rel.references)...))
	default:
		panic("unknown relation type")
	}

	expBuilder, err := b.buildFilterExp(fq, rf, filters)
	if err != nil {
		return nil, err
	}
	fq.SelectDataset = fq.Where(expBuilder)
	return fq, nil
}
