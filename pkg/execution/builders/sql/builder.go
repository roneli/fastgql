package sql

import (
	"fmt"
	"strings"

	"github.com/roneli/fastgql/internal/log"
	"github.com/roneli/fastgql/pkg/execution/builders"
	"github.com/roneli/fastgql/pkg/schema"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/iancoleman/strcase"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"github.com/vektah/gqlparser/v2/ast"

	// import the pg dialect
	_ "github.com/doug-martin/goqu/v9/dialect/postgres"
)

type defaultTableNameGenerator struct{}

func (tb defaultTableNameGenerator) Generate(_ int) string {
	return builders.GenerateTableName(6)
}

type Builder struct {
	Schema              *ast.Schema
	Logger              log.Logger
	TableNameGenerator  builders.TableNameGenerator
	Operators           map[string]builders.Operator
	AggregatorOperators map[string]builders.AggregatorOperator
	CaseConverter       builders.ColumnCaseConverter
}

func NewBuilder(config *builders.Config) Builder {
	var l log.Logger = log.NullLogger{}
	if config.Logger != nil {
		l = config.Logger
	}
	var tableNameGenerator builders.TableNameGenerator = defaultTableNameGenerator{}
	if config.TableNameGenerator != nil {
		tableNameGenerator = config.TableNameGenerator
	}
	var caseConverter builders.ColumnCaseConverter = strcase.ToSnake
	if config.ColumnCaseConverter != nil {
		caseConverter = config.ColumnCaseConverter
	}

	operators := make(map[string]builders.Operator)
	for k, v := range defaultOperators {
		operators[k] = v
	}
	for k, v := range config.CustomOperators {
		operators[k] = v
	}

	return Builder{Schema: config.Schema, Logger: l, TableNameGenerator: tableNameGenerator, Operators: operators, AggregatorOperators: defaultAggregatorOperators, CaseConverter: caseConverter}
}

func (b Builder) Create(field builders.Field) (string, []interface{}, error) {
	tableDef := getTableNamePrefix(b.Schema, "create", field.Field)
	input, ok := field.Arguments[builders.InputFieldName]
	if !ok {
		return "", nil, errors.New("missing input argument for create")
	}
	kv, err := getInputValues(input)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get input values: %w", err)
	}
	insertQuery, err := b.buildInsert(tableDef, kv)
	if err != nil {
		return "", nil, fmt.Errorf("failed to build delete query: %w", err)
	}
	withTable := goqu.T(b.CaseConverter(field.Name))
	// Generate payload response
	var cols []interface{}
	for _, f := range field.Selections {
		if f.Name == "rows_affected" {
			cols = append(cols, goqu.Select(goqu.COUNT(goqu.Star()).As("rows_affected")).From(withTable))
			continue
		}
		qh, err := b.buildQuery(tableDefinition{name: b.CaseConverter(field.Name)}, f)
		if err != nil {
			return "", nil, errors.New("failed to build payload data query")
		}
		cols = append(cols, qh.SelectJsonAgg(f.Name))
	}
	sql, args, err := goqu.Select(cols...).With(withTable.GetTable(), insertQuery).ToSQL()
	b.Logger.Debug("created insert query", "query", sql, "args", args, "error", err)
	return sql, args, err
}

func (b Builder) Delete(field builders.Field) (string, []interface{}, error) {
	tableDef := getTableNamePrefix(b.Schema, "delete", field.Field)
	deleteQuery, err := b.buildDelete(tableDef, field)
	if err != nil {
		return "", nil, fmt.Errorf("failed to build delete query: %w", err)
	}
	withTable := goqu.T(b.CaseConverter(field.Name))
	// Generate payload response
	var cols []interface{}
	for _, f := range field.Selections {
		if f.Name == "rows_affected" {
			cols = append(cols, goqu.Select(goqu.COUNT(goqu.Star()).As("rows_affected")).From(withTable))
			continue
		}
		qh, err := b.buildQuery(tableDefinition{name: b.CaseConverter(field.Name)}, f)
		if err != nil {
			return "", nil, errors.New("failed to build payload data query")
		}
		cols = append(cols, qh.SelectJsonAgg(f.Name))
	}
	sql, args, err := goqu.Select(cols...).With(withTable.GetTable(), deleteQuery).ToSQL()
	b.Logger.Debug("created delete query", "query", sql, "args", args, "error", err)
	return sql, args, err
}

func (b Builder) Update(field builders.Field) (string, []interface{}, error) {
	tableDef := getTableNamePrefix(b.Schema, "update", field.Field)
	updateQuery, err := b.buildUpdate(tableDef, field)
	if err != nil {
		return "", nil, err
	}
	withTable := goqu.T(b.CaseConverter(field.Name))
	// Generate payload response
	var cols []interface{}
	for _, f := range field.Selections {
		if f.Name == "rows_affected" {
			cols = append(cols, goqu.Select(goqu.COUNT(goqu.Star()).As("rows_affected")).From(withTable))
			continue
		}
		qh, err := b.buildQuery(tableDefinition{name: b.CaseConverter(field.Name)}, f)
		if err != nil {
			return "", nil, errors.New("failed to build payload data query")
		}
		cols = append(cols, qh.SelectJsonAgg(f.Name))
	}
	sql, args, err := goqu.Select(cols...).With(withTable.GetTable(), updateQuery).ToSQL()
	b.Logger.Debug("created update query", "query", sql, "args", args, "error", err)
	return sql, args, err
}

func (b Builder) Query(field builders.Field) (string, []interface{}, error) {
	var (
		query *queryHelper
		err   error
	)
	if strings.HasSuffix(field.Name, "Aggregate") && strings.HasPrefix(field.Name, "_") {
		query, err = b.buildAggregate(getAggregateTableName(b.Schema, field.Field), field)
	} else {
		query, err = b.buildQuery(getTableNameFromField(b.Schema, field.Definition), field)
	}
	if err != nil {
		return "", nil, err
	}
	q, args, err := query.SelectRow(true).ToSQL()
	b.Logger.Debug("created query", "query", q, "args", args, "error", err)
	return q, args, err
}

func (b Builder) buildUpdate(tableDef tableDefinition, field builders.Field) (*goqu.UpdateDataset, error) {
	b.Logger.Debug("building update", "tableDefinition", tableDef.name)
	tableAlias := b.TableNameGenerator.Generate(6)
	input, ok := field.Arguments["input"]
	if !ok {
		return nil, errors.New("missing input argument for update")
	}
	kv, err := getInputValues(input)
	if err != nil {
		return nil, fmt.Errorf("failed to get input values: %w", err)
	}
	// Substitute KV from GraphQL input into case conversion expected in database
	newRecord := make(map[string]interface{})
	for k, v := range kv[0] {
		newRecord[b.CaseConverter(k)] = v
	}
	table := tableDef.TableExpression().As(tableAlias)
	q := goqu.Dialect("postgres").Update(table).Set(newRecord).Prepared(true).Returning(goqu.Star())
	// if not filter is defined we will just return the query
	filterArg, ok := field.Arguments["filter"]
	if !ok {
		return q, nil
	}
	filters, ok := filterArg.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("expected filters map got %T", filterArg)
	}
	// Before we build the filter expression we require to get the Type definition
	// use the table be set Alias as the Alias used in the Update query
	filterExp, _ := b.buildFilterExp(tableHelper{table: tableDef.TableExpression().As(tableAlias)}, tableDef.objType, filters)
	return q.Where(filterExp), nil
}

func (b Builder) buildInsert(tableDef tableDefinition, kv []map[string]interface{}) (*goqu.InsertDataset, error) {
	b.Logger.Debug("building insert", "tableDefinition", tableDef.name)
	tableAlias := b.TableNameGenerator.Generate(6)
	table := tableDef.TableExpression().As(tableAlias)
	// Substitute KV from GraphQL input into case conversion expected in database
	for i, record := range kv {
		newRecord := make(map[string]interface{})
		for k, v := range record {
			newRecord[b.CaseConverter(k)] = v
		}
		kv[i] = newRecord
	}
	return goqu.Dialect("postgres").Insert(table).Rows(kv).Prepared(true).Returning(goqu.Star()), nil
}

func (b Builder) buildDelete(tableDef tableDefinition, field builders.Field) (*goqu.DeleteDataset, error) {
	b.Logger.Debug("building delete", "tableDefinition", tableDef.name)
	q := goqu.Dialect("postgres").Delete(tableDef.TableExpression()).Returning(goqu.Star())
	filterArg, ok := field.Arguments["filter"]
	// if not filter is defined we will just return the query
	if !ok {
		return q, nil
	}
	filters, ok := filterArg.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("expected filters map got %T", filterArg)
	}
	// Before we build the filter expression we require to get the Type definition
	filterExp, _ := b.buildFilterExp(tableHelper{table: tableDef.TableExpression().As(tableDef.name), alias: ""}, tableDef.objType, filters)
	return q.Where(filterExp), nil
}

func (b Builder) buildQuery(tableDef tableDefinition, field builders.Field) (*queryHelper, error) {
	b.Logger.Debug("building query", map[string]interface{}{"tableDefinition": tableDef.name})
	tableAlias := b.TableNameGenerator.Generate(6)
	table := tableDef.TableExpression().As(tableAlias)
	query := queryHelper{goqu.From(table), table, tableAlias, nil}

	fieldsAdded := make(map[string]struct{})
	// Add field columns
	for _, childField := range field.Selections {
		if _, ok := fieldsAdded[childField.Name]; ok {
			continue
		}
		fieldsAdded[childField.Name] = struct{}{}

		switch childField.FieldType {
		case builders.TypeScalar:
			b.Logger.Debug("adding field", "tableDefinition", tableDef.name, "fieldName", childField.Name)
			query.selects = append(query.selects, column{table: query.alias, name: b.CaseConverter(childField.Name), alias: childField.Name})
		case builders.TypeRelation:
			b.Logger.Debug("adding relation field", "tableDefinition", tableDef.name, "fieldName", childField.Name)
			if err := b.buildRelation(&query, childField); err != nil {
				return nil, fmt.Errorf("failed to build relation for %s", childField.Name)
			}
		case builders.TypeAggregate:
			if err := b.buildRelationAggregate(&query, childField); err != nil {
				return nil, fmt.Errorf("failed to build relation for %s", childField.Name)
			}
		default:
			b.Logger.Error("unknown field type", "tableDefinition", tableDef.name, "fieldName", childField.Name, "fieldType", childField.FieldType)
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

func (b Builder) buildAggregate(tableDef tableDefinition, field builders.Field) (*queryHelper, error) {
	b.Logger.Debug("building aggregate", "tableDefinition", tableDef.name)
	tableAlias := b.TableNameGenerator.Generate(6)
	table := tableDef.TableExpression().As(tableAlias)
	query := &queryHelper{goqu.From(table), table, tableAlias, nil}

	var aggColumns []interface{}
	for _, f := range field.Selections {
		switch f.Name {
		case "count":
			aggColumns = append(aggColumns, goqu.L("'count'"), goqu.COUNT(goqu.L("1")))
		default:
			if op, ok := b.AggregatorOperators[f.Name]; ok {
				aggExp, err := op(table, f.Selections)
				if err != nil {
					return nil, err
				}
				aggColumns = append(aggColumns, goqu.L(fmt.Sprintf("'%s'", f.Name)), aggExp)

			} else {
				return nil, fmt.Errorf("aggrgator %s not supported", f.Name)
			}
		}
	}
	if err := b.buildFiltering(query, field); err != nil {
		return nil, err
	}

	query.SelectDataset = query.Select(goqu.Func("json_build_object", aggColumns...).As(field.Name))
	return query, nil
}

func (b Builder) buildOrdering(query *queryHelper, field builders.Field) {
	orderBy, ok := field.Arguments["orderBy"]
	if !ok {
		return
	}
	orderFields, _ := builders.CollectOrdering(orderBy)

	for _, o := range orderFields {
		b.Logger.Debug("adding ordering", "tableDefinition", query.TableName(), "field", o.Key, "orderType", o.Type)
		switch o.Type {
		case builders.OrderingTypesAsc:
			query.SelectDataset = query.OrderAppend(goqu.C(b.CaseConverter(o.Key)).Asc().NullsLast())
		case builders.OrderingTypesAscNull:
			query.SelectDataset = query.OrderAppend(goqu.C(b.CaseConverter(o.Key)).Asc().NullsFirst())
		case builders.OrderingTypesDesc:
			query.SelectDataset = query.OrderAppend(goqu.C(b.CaseConverter(o.Key)).Desc().NullsLast())
		case builders.OrderingTypesDescNull:
			query.SelectDataset = query.OrderAppend(goqu.C(b.CaseConverter(o.Key)).Desc().NullsFirst())
		}
	}
}

func (b Builder) buildPagination(query *queryHelper, field builders.Field) {
	if limit, ok := field.Arguments["limit"]; ok {
		b.Logger.Debug("adding pagination limit", "tableDefinition", query.TableName(), "limit", limit)
		query.SelectDataset = query.Limit(cast.ToUint(limit))
	}
	if offset, ok := field.Arguments["offset"]; ok {
		b.Logger.Debug("adding pagination offset", "tableDefinition", query.TableName(), "offset", offset)
		query.SelectDataset = query.Offset(cast.ToUint(offset))
	}
}

func (b Builder) buildFiltering(query *queryHelper, field builders.Field) error {
	filterArg, ok := field.Arguments["filter"]
	if !ok {
		return nil
	}
	filters, ok := filterArg.(map[string]interface{})
	if !ok {
		return fmt.Errorf("unexpected filter arg type")
	}
	filterExp, _ := b.buildFilterExp(query.Table(), field.TypeDefinition, filters)
	query.SelectDataset = query.Where(filterExp)
	return nil
}

func (b Builder) buildFilterLogicalExp(table tableHelper, astDefinition *ast.Definition, filtersList []interface{}, logicalType exp.ExpressionListType) (goqu.Expression, error) {
	expBuilder := exp.NewExpressionList(logicalType)
	for _, filterValue := range filtersList {
		kv, ok := filterValue.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("fatal value of bool exp not map")
		}
		filterExp, err := b.buildFilterExp(table, astDefinition, kv)
		if err != nil {
			return nil, err
		}
		expBuilder = expBuilder.Append(filterExp)
	}
	return expBuilder, nil
}

func (b Builder) buildFilterExp(table tableHelper, astDefinition *ast.Definition, filters map[string]interface{}) (goqu.Expression, error) {
	filterInputDef := builders.GetFilterInput(b.Schema, astDefinition)
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
			logicalExp, err := b.buildFilterLogicalExp(table, astDefinition, vv, logicalType)
			if err != nil {
				return nil, err
			}
			expBuilder = expBuilder.Append(logicalExp)
		case k == string(schema.LogicalOperatorNot):
			filterExp, err := b.buildFilterExp(table, astDefinition, filters)
			if err != nil {
				return nil, err
			}
			expBuilder = expBuilder.Append(goqu.Func("NOT", filterExp))
		case strings.HasSuffix(keyType.Name(), "FilterInput"):
			kv, ok := v.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("fatal value of bool exp not map")
			}
			ffd := astDefinition.Fields.ForName(k)
			// Create a Builder
			d := ffd.Directives.ForName("relation")
			if d == nil {
				return nil, fmt.Errorf("missing directive sqlRelation")
			}
			fq, err := b.buildFilterQuery(table, b.Schema.Types[ffd.Type.Name()], parseRelationDirective(d), kv)
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
				opExp, err := b.Operation(table.table, k, op, value)
				if err != nil {
					return nil, err
				}
				expBuilder = expBuilder.Append(opExp)
			}
		}
	}
	return expBuilder, nil
}

func (b Builder) Operation(table exp.AliasedExpression, fieldName, operatorName string, value interface{}) (goqu.Expression, error) {
	opFunc, ok := b.Operators[operatorName]
	if !ok {
		return nil, fmt.Errorf("key operator %s not supported", operatorName)
	}
	return opFunc(table, b.CaseConverter(fieldName), value), nil
}

func (b Builder) buildRelation(parentQuery *queryHelper, rf builders.Field) error {
	tableDef := getTableNameFromField(b.Schema, rf.Definition)
	relationQuery, err := b.buildQuery(tableDef, rf)
	if err != nil {
		return errors.Wrap(err, "failed building relation")
	}
	rel := parseRelationDirective(rf.Definition.Directives.ForName("relation"))
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
		m2mTableAlias := b.TableNameGenerator.Generate(6)
		m2mTable := goqu.T(rel.manyToManyTable).Schema(tableDef.schema).As(m2mTableAlias)
		m2mQuery := queryHelper{
			SelectDataset: goqu.From(m2mTable),
			table:         m2mTable,
			alias:         m2mTableAlias,
			selects:       relationQuery.selects,
		}
		// Join m2mBuilder with the relBuilder
		m2mQuery.SelectDataset = m2mQuery.LeftJoin(
			goqu.Lateral(relationQuery.SelectRow(false).Where(buildCrossCondition(relationQuery.alias, rel.references, m2mTableAlias, rel.manyToManyReferences))).As(relationQuery.alias),
			goqu.On(goqu.L("true")))

		// Add cross condition from parent Builder (current Builder instance)
		m2mQuery.SelectDataset = m2mQuery.Where(buildCrossCondition(parentQuery.alias, rel.fields, m2mTableAlias, rel.manyToManyFields)).As(relationQuery.alias)

		// Finally, aggregate relation query and join the m2m tableDefinition with the main query
		aggTableName := b.TableNameGenerator.Generate(6)
		aggQuery := goqu.From(m2mQuery.SelectRow(false)).As(aggTableName).Select(relationQuery.buildJsonAgg(rf.Name).As(rf.Name)).As(aggTableName).Where(goqu.T(relationQuery.alias).IsNot(nil))
		parentQuery.SelectDataset = parentQuery.CrossJoin(goqu.Lateral(aggQuery))
		parentQuery.selects = append(parentQuery.selects, column{name: rf.Name, alias: "", table: aggTableName})

	}
	return nil
}

func (b Builder) buildRelationAggregate(parentQuery *queryHelper, rf builders.Field) error {
	// Build aggregate query
	aggQuery, err := b.buildAggregate(getAggregateTableName(b.Schema, rf.Field), rf)
	if err != nil {
		return errors.Wrap(err, "failed building relation")
	}

	originalDef := rf.ObjectDefinition.Fields.ForName(strings.Split(rf.Name, "Aggregate")[0][1:])
	rel := parseRelationDirective(originalDef.Directives.ForName("sqlRelation"))
	switch rel.relType {
	case OneToMany:
		parentQuery.SelectDataset = parentQuery.SelectDataset.LeftJoin(
			goqu.Lateral(aggQuery.As(aggQuery.alias).
				Where(buildCrossCondition(parentQuery.alias, rel.fields, aggQuery.alias, rel.references))),
			goqu.On(goqu.L("true")),
		)
		parentQuery.selects = append(parentQuery.selects, column{name: rf.Name, alias: "", table: aggQuery.alias})
	case ManyToMany:
		m2mTableName := b.TableNameGenerator.Generate(6)
		jExps := buildJoinCondition(parentQuery.alias, rel.fields, m2mTableName, rel.manyToManyFields)
		jExps = append(jExps, buildJoinCondition(m2mTableName, rel.manyToManyReferences, aggQuery.alias, rel.references)...)
		aggQuery.SelectDataset = aggQuery.InnerJoin(goqu.T(rel.manyToManyTable).As(m2mTableName), goqu.On(jExps...))
		parentQuery.SelectDataset = parentQuery.CrossJoin(goqu.Lateral(aggQuery).As(aggQuery.alias))
		parentQuery.selects = append(parentQuery.selects, column{name: rf.Name, alias: "", table: aggQuery.alias})
	}
	return nil
}

func (b Builder) buildFilterQuery(parentTable tableHelper, rf *ast.Definition, rel relation, filters map[string]interface{}) (*queryHelper, error) {
	tableAlias := b.TableNameGenerator.Generate(6)
	td, err := builders.GetTableDirective(rf)
	if err != nil {
		return nil, fmt.Errorf("missing @table directive to create filter query for %s: %w", rf.Name, err)
	}
	table := goqu.T(td.Name).Schema(td.Schema).As(tableAlias)
	fq := &queryHelper{goqu.From(table), table, tableAlias, nil}

	switch rel.relType {
	case ManyToMany:
		m2mTableName := b.TableNameGenerator.Generate(6)
		jExps := buildJoinCondition(parentTable.alias, rel.fields, m2mTableName, rel.manyToManyFields)
		jExps = append(jExps, buildJoinCondition(m2mTableName, rel.manyToManyReferences, fq.alias, rel.references)...)
		fq.SelectDataset = fq.InnerJoin(goqu.T(rel.manyToManyTable).Schema(td.Schema).As(m2mTableName), goqu.On(jExps...))
	case OneToOne:
		relationTableName := b.TableNameGenerator.Generate(6)
		jExps := buildJoinCondition(parentTable.alias, rel.fields, fq.alias, rel.references)
		jExps = append(jExps, buildJoinCondition(parentTable.alias, rel.fields, relationTableName, rel.fields)...)
		fq.SelectDataset = fq.InnerJoin(goqu.T(rel.baseTable).Schema(td.Schema).As(relationTableName), goqu.On(jExps...))
	case OneToMany:
		fq.SelectDataset = fq.InnerJoin(goqu.T(rel.baseTable).Schema(td.Schema).As(b.TableNameGenerator.Generate(6)),
			goqu.On(buildJoinCondition(parentTable.alias, rel.fields, fq.alias, rel.references)...))
	default:
		panic("unknown relation type")
	}

	expBuilder, err := b.buildFilterExp(fq.Table(), rf, filters)
	if err != nil {
		return nil, err
	}
	fq.SelectDataset = fq.Where(expBuilder)
	return fq, nil
}
