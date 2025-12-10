package sql

import (
	"fmt"
	"math/rand"
	"slices"
	"strings"
	"unsafe"

	"github.com/roneli/fastgql/pkg/schema"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/iancoleman/strcase"
	"github.com/pkg/errors"
	"github.com/roneli/fastgql/pkg/execution/builders"
	"github.com/roneli/fastgql/pkg/log"
	"github.com/spf13/cast"
	"github.com/vektah/gqlparser/v2/ast"

	// import the pg dialect
	_ "github.com/doug-martin/goqu/v9/dialect/postgres"
)

const (
	letterBytes   = "abcdefghijklmnopqrstuvwxyz"
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

type defaultTableNameGenerator struct{}

func (tb defaultTableNameGenerator) Generate(_ int) string {
	return generateTableName(6)
}

func generateTableName(n int) string {
	b := make([]byte, n)
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

type Builder struct {
	Schema              *ast.Schema
	Logger              log.Logger
	TableNameGenerator  builders.TableNameGenerator
	Operators           map[string]builders.Operator
	AggregatorOperators map[string]builders.AggregatorOperator
	CaseConverter       builders.ColumnCaseConverter
	Dialect             string
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

	dialect := config.Dialect
	if dialect == "" {
		dialect = "postgres" // Default to PostgreSQL for backwards compatibility
	}

	operators := make(map[string]builders.Operator)
	for k, v := range defaultOperators {
		operators[k] = v
	}
	for k, v := range config.CustomOperators {
		operators[k] = v
	}

	return Builder{
		Schema:              config.Schema,
		Logger:              l,
		TableNameGenerator:  tableNameGenerator,
		Operators:           operators,
		AggregatorOperators: defaultAggregatorOperators,
		CaseConverter:       caseConverter,
		Dialect:             dialect,
	}
}

// Capabilities returns what this SQL database supports.
func (b Builder) Capabilities() builders.Capabilities {
	return builders.Capabilities{
		SupportsJoins:        true,
		SupportsReturning:    true,
		SupportsTransactions: true,
		MaxRelationDepth:     -1, // unlimited
	}
}

// Create generates an SQL create query based on graphql ast.
func (b Builder) Create(field builders.Field) (string, []any, error) {
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
	q, err := b.buildPayloadQuery(withTable, insertQuery, field)
	if err != nil {
		return "", nil, err
	}
	sql, args, err := q.ToSQL()
	b.Logger.Debug("created insert query", "query", sql, "args", args, "error", err)
	return sql, args, err
}

// Delete generates an SQL delete query based on graphql ast.
func (b Builder) Delete(field builders.Field) (string, []any, error) {
	tableDef := getTableNamePrefix(b.Schema, "delete", field.Field)
	deleteQuery, err := b.buildDelete(tableDef, field)
	if err != nil {
		return "", nil, fmt.Errorf("failed to build delete query: %w", err)
	}
	withTable := goqu.T(b.CaseConverter(field.Name))
	// Generate payload response
	q, err := b.buildPayloadQuery(withTable, deleteQuery, field)
	if err != nil {
		return "", nil, err
	}
	sql, args, err := q.ToSQL()
	b.Logger.Debug("created delete query", "query", sql, "args", args, "error", err)
	return sql, args, err
}

// Update generates an SQL update query based on graphql ast.
func (b Builder) Update(field builders.Field) (string, []any, error) {
	tableDef := getTableNamePrefix(b.Schema, "update", field.Field)
	updateQuery, err := b.buildUpdate(tableDef, field)
	if err != nil {
		return "", nil, err
	}
	withTable := goqu.T(b.CaseConverter(field.Name))
	// Generate payload response
	q, err := b.buildPayloadQuery(withTable, updateQuery, field)
	if err != nil {
		return "", nil, err
	}
	sql, args, err := q.ToSQL()
	b.Logger.Debug("created update query", "query", sql, "args", args, "error", err)
	return sql, args, err
}

// Query generates an SQL read query based on graphql ast.
func (b Builder) Query(field builders.Field) (string, []any, error) {
	var (
		query *queryHelper
		err   error
	)
	if strings.HasSuffix(field.Name, "Aggregate") && strings.HasPrefix(field.Name, "_") {
		// alias in root level
		query, err = b.buildAggregate(getAggregateTableName(b.Schema, field.Field), field, true)
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

// ======================================= Helper Methods ================================================== //

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
	newRecord := make(map[string]any)
	for k, v := range kv[0] {
		newRecord[b.CaseConverter(k)] = v
	}
	table := tableDef.TableExpression().As(tableAlias)
	q := goqu.Dialect(b.Dialect).Update(table).Set(newRecord).Prepared(true).Returning(goqu.Star())
	// if not filter is defined we will just return the query
	filterArg, ok := field.Arguments["filter"]
	if !ok {
		return q, nil
	}
	filters, ok := filterArg.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected filters map got %T", filterArg)
	}
	// Before we build the filter expression we require to get the Type definition
	// use the table be set Alias as the Alias used in the Update query
	filterExp, _ := b.buildFilterExp(tableHelper{table: tableDef.TableExpression().As(tableAlias)}, tableDef.objType, filters)
	return q.Where(filterExp), nil
}

func (b Builder) buildInsert(tableDef tableDefinition, kv []map[string]any) (*goqu.InsertDataset, error) {
	b.Logger.Debug("building insert", "tableDefinition", tableDef.name)
	tableAlias := b.TableNameGenerator.Generate(6)
	table := tableDef.TableExpression().As(tableAlias)
	// Substitute KV from GraphQL input into case conversion expected in database
	for i, record := range kv {
		newRecord := make(map[string]any)
		for k, v := range record {
			newRecord[b.CaseConverter(k)] = v
		}
		kv[i] = newRecord
	}
	return goqu.Dialect(b.Dialect).Insert(table).Rows(kv).Prepared(true).Returning(goqu.Star()), nil
}

func (b Builder) buildDelete(tableDef tableDefinition, field builders.Field) (*goqu.DeleteDataset, error) {
	b.Logger.Debug("building delete", "tableDefinition", tableDef.name)
	q := goqu.Dialect(b.Dialect).Delete(tableDef.TableExpression()).Returning(goqu.Star())
	filterArg, ok := field.Arguments["filter"]
	// if not filter is defined we will just return the query
	if !ok {
		return q, nil
	}
	filters, ok := filterArg.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected filters map got %T", filterArg)
	}
	// Before we build the filter expression we require to get the Type definition
	filterExp, _ := b.buildFilterExp(tableHelper{table: tableDef.TableExpression().As(tableDef.name), alias: ""}, tableDef.objType, filters)
	return q.Where(filterExp), nil
}

func (b Builder) buildQuery(tableDef tableDefinition, field builders.Field) (*queryHelper, error) {
	b.Logger.Debug("building query", map[string]any{"tableDefinition": tableDef.name})
	tableAlias := b.TableNameGenerator.Generate(6)
	table := tableDef.TableExpression().As(tableAlias)
	query := queryHelper{goqu.From(table), table, tableAlias, nil, b.Dialect}

	fieldsAdded := make(map[string]struct{})
	// if type is abstract check if it has a typename
	if field.TypeDefinition.IsAbstractType() {
		// add type name field
		d := field.TypeDefinition.Directives.ForName("typename")
		if d != nil {
			name := d.Arguments.ForName("name")
			query.selects = append(query.selects, column{table: query.alias, name: b.CaseConverter(name.Value.Raw), alias: name.Value.Raw})
			b.Logger.Debug("adding typename field for interface", "interface", field.TypeDefinition.Name, "tableDefinition", tableDef.name, "fieldName", name.Value.Raw)
			fieldsAdded[name.Value.Raw] = struct{}{}
		}
	}
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

func (b Builder) buildPayloadQuery(withTable exp.IdentifierExpression, baseQuery exp.Expression, field builders.Field) (*goqu.SelectDataset, error) {
	// Generate payload response
	cols := make([]any, 0, len(field.Selections))
	hasRowsAffected := false
	for _, f := range field.Selections {
		if f.Name == "rows_affected" {
			hasRowsAffected = true
			continue
		}
		qh, err := b.buildQuery(tableDefinition{name: b.CaseConverter(field.Name)}, f)
		if err != nil {
			return nil, errors.New("failed to build payload data query")
		}
		cols = append(cols, qh.SelectJsonAgg(f.Name))
	}
	if hasRowsAffected {
		cols = append(cols, goqu.Select(goqu.COUNT(goqu.Star()).As("rows_affected")).From(withTable))
	}
	return goqu.Select(cols...).With(withTable.GetTable(), baseQuery), nil
}

func (b Builder) buildAggregateGroupBy(table exp.AliasedExpression, groupBy []string) ([]any, []any) {
	groupByCols := make([]any, 0, len(groupBy))
	groupByResult := make([]any, 0, 2*len(groupBy))
	for _, k := range groupBy {
		groupByCols = append(groupByCols, table.Col(b.CaseConverter(k)))
		groupByResult = append(groupByResult, goqu.L(fmt.Sprintf("'%s'", b.CaseConverter(k))), table.Col(b.CaseConverter(k)))
	}
	return groupByCols, groupByResult

}

func (b Builder) buildAggregate(tableDef tableDefinition, field builders.Field, aliasAggregates bool) (*queryHelper, error) {
	b.Logger.Debug("building aggregate", "tableDefinition", tableDef.name)
	tableAlias := b.TableNameGenerator.Generate(6)
	table := tableDef.TableExpression().As(tableAlias)
	query := &queryHelper{goqu.Dialect(b.Dialect).From(table), table, tableAlias, nil, b.Dialect}
	var fieldExp exp.Expression
	for _, f := range field.Selections {
		switch f.Name {
		case "group":
			groupBy, ok := field.Arguments["groupBy"]
			if !ok {
				continue
			}
			groupByMap, err := cast.ToStringSliceE(groupBy)
			if err != nil {
				return nil, fmt.Errorf("expected group by map got %T", groupBy)
			}
			groupByCols, groupByResult := b.buildAggregateGroupBy(table, groupByMap)
			fieldExp = goqu.Func("json_build_object", groupByResult...)
			if aliasAggregates {
				fieldExp = fieldExp.(exp.Aliaseable).As("group")
			}
			query.selects = append(query.selects, column{table: query.alias, name: f.Name, expression: fieldExp})
			query.SelectDataset = query.GroupBy(groupByCols...)
		case "count":
			b.Logger.Debug("adding field", "tableDefinition", tableDef.name, "fieldName", f.Name)
			fieldExp = goqu.COUNT(goqu.L("1"))
			if aliasAggregates {
				fieldExp = fieldExp.(exp.Aliaseable).As(f.Name)
			}
			query.selects = append(query.selects, column{table: query.alias, name: f.Name, alias: f.Name, expression: fieldExp})
		default:
			if op, ok := b.AggregatorOperators[f.Name]; ok {
				aggExp, err := op(table, f.Selections)
				if err != nil {
					return nil, err
				}
				if aliasAggregates {
					aggExp = aggExp.(exp.Aliaseable).As(f.Name)
				}
				query.selects = append(query.selects, column{table: query.alias, name: f.Name, expression: aggExp})
			} else {
				return nil, fmt.Errorf("aggrgator %s not supported", f.Name)
			}
		}
	}
	if err := b.buildFiltering(query, field); err != nil {
		return nil, err
	}
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
	filters, ok := filterArg.(map[string]any)
	if !ok {
		return fmt.Errorf("unexpected filter arg type")
	}
	filterExp, _ := b.buildFilterExp(query.Table(), field.TypeDefinition, filters)
	query.SelectDataset = query.Where(filterExp)
	return nil
}

func (b Builder) buildFilterLogicalExp(table tableHelper, astDefinition *ast.Definition, filtersList []any, logicalType exp.ExpressionListType) (goqu.Expression, error) {
	expBuilder := exp.NewExpressionList(logicalType)
	for _, filterValue := range filtersList {
		kv, ok := filterValue.(map[string]any)
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

func (b Builder) buildFilterExp(table tableHelper, astDefinition *ast.Definition, filters map[string]any) (goqu.Expression, error) {
	filterInputDef := builders.GetFilterInput(b.Schema, astDefinition)
	expBuilder := exp.NewExpressionList(exp.AndType)

	// Sort filter keys for deterministic SQL generation
	filterKeys := make([]string, 0, len(filters))
	for k := range filters {
		filterKeys = append(filterKeys, k)
	}
	slices.Sort(filterKeys)

	for _, k := range filterKeys {
		v := filters[k]
		keyType := filterInputDef.Fields.ForName(k).Type
		switch {
		case k == string(builders.LogicalOperatorAND) || k == string(builders.LogicalOperatorOR):
			vv, ok := v.([]any)
			if !ok {
				return nil, fmt.Errorf("fatal value of logical list exp not list")
			}
			logicalType := exp.AndType
			if k == string(builders.LogicalOperatorOR) {
				logicalType = exp.OrType
			}
			logicalExp, err := b.buildFilterLogicalExp(table, astDefinition, vv, logicalType)
			if err != nil {
				return nil, err
			}
			expBuilder = expBuilder.Append(logicalExp)
		case k == string(builders.LogicalOperatorNot):
			filterExp, err := b.buildFilterExp(table, astDefinition, filters)
			if err != nil {
				return nil, err
			}
			expBuilder = expBuilder.Append(goqu.Func("NOT", filterExp))
		case keyType.Name() == "MapComparator":
			// Handle Map scalar filtering with JSONPath
			kv, ok := v.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("MapComparator value must be a map")
			}
			filter, err := ParseMapComparator(kv)
			if err != nil {
				return nil, fmt.Errorf("parsing MapComparator for %s: %w", k, err)
			}
			col := table.table.Col(b.CaseConverter(k))
			jsonExp, err := BuildMapFilter(col, filter)
			if err != nil {
				return nil, fmt.Errorf("building JSON filter for %s: %w", k, err)
			}
			expBuilder = expBuilder.Append(jsonExp)
		case strings.HasSuffix(keyType.Name(), "FilterInput"):
			kv, ok := v.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("fatal value of bool exp not map")
			}
			fid := filterInputDef.Fields.ForName(k)
			if fid.Directives.ForName("isInterfaceFilter") != nil {
				// add type filter + interface filter
				expBuilder = expBuilder.Append(b.buildInterfaceFilter(table, astDefinition, b.Schema.Types[strcase.ToCamel(k)], kv))
				continue
			}

			ffd := astDefinition.Fields.ForName(k)

			// Check if field has @json directive - use JSONPath filter instead of EXISTS subquery
			if jsonDir := ffd.Directives.ForName("json"); jsonDir != nil {
				col := table.table.Col(b.CaseConverter(k))
				jsonPath, vars, err := BuildJsonFilterFromOperatorMap(kv)
				if err != nil {
					return nil, fmt.Errorf("building JSON filter for @json field %s: %w", k, err)
				}
				jsonExp, err := BuildJsonPathExistsExpression(col, jsonPath, vars)
				if err != nil {
					return nil, fmt.Errorf("building JSONPath expression for %s: %w", k, err)
				}
				expBuilder = expBuilder.Append(jsonExp)
				continue
			}

			// Create a Builder for relational filter
			rel := schema.GetRelationDirective(ffd)
			if rel == nil {
				return nil, fmt.Errorf("missing directive relation")
			}
			fq, err := b.buildFilterQuery(table, b.Schema.Types[ffd.Type.Name()], *rel, kv)
			if err != nil {
				return nil, err
			}
			expBuilder = expBuilder.Append(goqu.Func("exists", fq.SelectOne()))
		default:
			opMap, ok := v.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("fatal value of key not map")
			}
			// Sort operator keys for deterministic SQL generation
			opKeys := make([]string, 0, len(opMap))
			for op := range opMap {
				opKeys = append(opKeys, op)
			}
			slices.Sort(opKeys)

			for _, op := range opKeys {
				value := opMap[op]
				opExp, err := b.buildOperation(table.table, k, op, value)
				if err != nil {
					return nil, err
				}
				expBuilder = expBuilder.Append(opExp)
			}
		}
	}
	return expBuilder, nil
}

func (b Builder) buildRelation(parentQuery *queryHelper, rf builders.Field) error {
	tableDef := getTableNameFromField(b.Schema, rf.Definition)
	relationQuery, err := b.buildQuery(tableDef, rf)
	if err != nil {
		return errors.Wrap(err, "failed building relation")
	}
	rel := schema.GetRelationDirective(rf.Definition)
	switch rel.RelType {
	case schema.OneToOne:
		parentQuery.SelectDataset = parentQuery.LeftJoin(goqu.Lateral(relationQuery.SelectJson(rf.Name).As(relationQuery.alias).
			Where(buildCrossCondition(parentQuery.alias, rel.Fields, relationQuery.alias, rel.References))),
			goqu.On(goqu.L("true")),
		)
		parentQuery.selects = append(parentQuery.selects, column{name: rf.Name, alias: "", table: relationQuery.alias})
	case schema.OneToMany:
		parentQuery.SelectDataset = parentQuery.LeftJoin(
			goqu.Lateral(relationQuery.SelectJsonAgg(rf.Name).As(relationQuery.alias).
				Where(buildCrossCondition(parentQuery.alias, rel.Fields, relationQuery.alias, rel.References))),
			goqu.On(goqu.L("true")),
		)
		parentQuery.selects = append(parentQuery.selects, column{name: rf.Name, alias: "", table: relationQuery.alias})
	case schema.ManyToMany:
		m2mTableAlias := b.TableNameGenerator.Generate(6)
		m2mTable := goqu.T(rel.ManyToManyTable).Schema(tableDef.schema).As(m2mTableAlias)
		m2mQuery := queryHelper{
			SelectDataset: goqu.From(m2mTable),
			table:         m2mTable,
			alias:         m2mTableAlias,
			selects:       relationQuery.selects,
			dialect:       b.Dialect,
		}
		// Join m2mBuilder with the relBuilder
		m2mQuery.SelectDataset = m2mQuery.LeftJoin(
			goqu.Lateral(relationQuery.SelectRow(false).Where(buildCrossCondition(relationQuery.alias, rel.References, m2mTableAlias, rel.ManyToManyReferences))).As(relationQuery.alias),
			goqu.On(goqu.L("true")))

		// Add cross condition from parent Builder (current Builder instance)
		m2mQuery.SelectDataset = m2mQuery.Where(buildCrossCondition(parentQuery.alias, rel.Fields, m2mTableAlias, rel.ManyToManyFields)).As(relationQuery.alias)

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
	aggQuery, err := b.buildAggregate(getAggregateTableName(b.Schema, rf.Field), rf, false)
	if err != nil {
		return errors.Wrap(err, "failed building relation")
	}
	originalDef := rf.ObjectDefinition.Fields.ForName(strings.Split(rf.Name, "Aggregate")[0][1:])
	rel := schema.GetRelationDirective(originalDef)
	name := b.CaseConverter(rf.Name)
	// TODO: finish this
	switch rel.RelType {
	case schema.OneToMany, schema.OneToOne:
		parentQuery.SelectDataset = parentQuery.LeftJoin(
			goqu.Lateral(goqu.Select(goqu.Func("jsonb_agg", aggQuery.table.Col(name)).As(name)).From(aggQuery.SelectJson(name).As(aggQuery.alias).
				Where(buildCrossCondition(parentQuery.alias, rel.Fields, aggQuery.alias, rel.References)))).As(aggQuery.alias),
			goqu.On(goqu.L("true")),
		)
		parentQuery.selects = append(parentQuery.selects, column{name: name, alias: "", table: aggQuery.alias})
	case schema.ManyToMany:
		m2mTableName := b.TableNameGenerator.Generate(6)
		jExps := buildJoinCondition(parentQuery.alias, rel.Fields, m2mTableName, rel.ManyToManyFields)
		jExps = append(jExps, buildJoinCondition(m2mTableName, rel.ManyToManyReferences, aggQuery.alias, rel.References)...)
		aggQuery.SelectDataset = aggQuery.InnerJoin(goqu.T(rel.ManyToManyTable).As(m2mTableName), goqu.On(jExps...))
		parentQuery.SelectDataset = parentQuery.CrossJoin(goqu.Lateral(goqu.Select(goqu.Func("jsonb_agg", aggQuery.table.Col(name)).As(name)).From(aggQuery.SelectJson(name).As(aggQuery.alias))).As(aggQuery.alias))
		parentQuery.selects = append(parentQuery.selects, column{name: b.CaseConverter(name), alias: "", table: aggQuery.alias})
	}
	return nil
}

func (b Builder) buildFilterQuery(parentTable tableHelper, rf *ast.Definition, rel schema.RelationDirective, filters map[string]any) (*queryHelper, error) {
	tableAlias := b.TableNameGenerator.Generate(6)
	td, err := schema.GetTableDirective(rf)
	if err != nil {
		return nil, fmt.Errorf("missing @table directive to create filter query for %s: %w", rf.Name, err)
	}
	table := goqu.T(td.Name).Schema(td.Schema).As(tableAlias)
	fq := &queryHelper{goqu.From(table), table, tableAlias, nil, b.Dialect}

	switch rel.RelType {
	case schema.ManyToMany:
		m2mTableName := b.TableNameGenerator.Generate(6)
		jExps := buildJoinCondition(parentTable.alias, rel.Fields, m2mTableName, rel.ManyToManyFields)
		jExps = append(jExps, buildJoinCondition(m2mTableName, rel.ManyToManyReferences, fq.alias, rel.References)...)
		fq.SelectDataset = fq.InnerJoin(goqu.T(rel.ManyToManyTable).Schema(td.Schema).As(m2mTableName), goqu.On(jExps...))
	case schema.OneToOne:
		relationTableName := b.TableNameGenerator.Generate(6)
		jExps := buildJoinCondition(parentTable.alias, rel.Fields, fq.alias, rel.References)
		jExps = append(jExps, buildJoinCondition(parentTable.alias, rel.Fields, relationTableName, rel.References)...)
		fq.SelectDataset = fq.InnerJoin(goqu.T(td.Name).Schema(td.Schema).As(relationTableName), goqu.On(jExps...))
	case schema.OneToMany:
		fq.SelectDataset = fq.InnerJoin(parentTable.table.Aliased().(exp.Aliaseable).As(b.TableNameGenerator.Generate(6)),
			goqu.On(buildJoinCondition(parentTable.alias, rel.Fields, fq.alias, rel.References)...))
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

func (b Builder) buildInterfaceFilter(table tableHelper, parentDef, definition *ast.Definition, kv map[string]any) goqu.Expression {
	d := parentDef.Directives.ForName("typename").Arguments.ForName("name").Value.Raw
	filterExp, err := b.buildFilterExp(table, definition, kv)
	if err != nil {
		panic(err)
	}
	return goqu.And(filterExp, table.table.Col(d).Eq(strings.ToLower(definition.Name)))
}

// buildOperation creates a goqu.Expression SQL operator
func (b Builder) buildOperation(table exp.AliasedExpression, fieldName, operatorName string, value any) (goqu.Expression, error) {
	opFunc, ok := b.Operators[operatorName]
	if !ok {
		return nil, fmt.Errorf("key operator %s not supported", operatorName)
	}
	return opFunc(table, b.CaseConverter(fieldName), value), nil
}
