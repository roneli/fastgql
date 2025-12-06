package sql_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/spf13/cast"

	"github.com/99designs/gqlgen/graphql"
	"github.com/roneli/fastgql/pkg/execution/builders"
	"github.com/roneli/fastgql/pkg/execution/builders/sql"
	"github.com/stretchr/testify/require"

	"github.com/roneli/fastgql/pkg/schema"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/parser"
	"github.com/vektah/gqlparser/v2/validator"
)

type TestBuilderCase struct {
	Name              string
	SchemaFile        string
	GraphQLQuery      string
	ExpectedArguments []interface{}
	ExpectedSQL       string
	CustomOperators   map[string]builders.Operator
}

type TestTableNameGenerator struct {
	Index int
}

func (t *TestTableNameGenerator) Generate(_ int) string {
	name := fmt.Sprintf("sq%d", t.Index)
	t.Index += 1
	return name
}

func (t *TestTableNameGenerator) Reset() {
	t.Index = 0
}

func TestBuilder_Query(t *testing.T) {
	testCases := []TestBuilderCase{
		{
			Name:              "base_query",
			SchemaFile:        "testdata/schema_simple.graphql",
			GraphQLQuery:      `query { users { name } }`,
			ExpectedSQL:       `SELECT "sq0"."name" AS "name" FROM "app"."users" AS "sq0" LIMIT $1`,
			ExpectedArguments: []interface{}{int64(100)},
		},
		{
			Name:              "query_with_limit",
			SchemaFile:        "testdata/schema_simple.graphql",
			GraphQLQuery:      `query { users(limit:5) { name } }`,
			ExpectedSQL:       `SELECT "sq0"."name" AS "name" FROM "app"."users" AS "sq0" LIMIT $1`,
			ExpectedArguments: []interface{}{int64(5)},
		},
		{
			Name:       "complex_query_with_two_level_filter",
			SchemaFile: "testdata/schema_simple.graphql",
			GraphQLQuery: `query {
							  users(limit: 5, filter: {posts: {categories: {name: {eq: "IT"}}}}) {
								name
								posts(filter: {name: {like: "%po%"}}) {
								  name
								}
							  }
							}`,
			ExpectedSQL:       `SELECT "sq0"."name" AS "name", "sq1"."posts" AS "posts" FROM "app"."users" AS "sq0" LEFT JOIN LATERAL (SELECT COALESCE(jsonb_agg(jsonb_build_object('name', "sq1"."name")), '[]'::jsonb) AS "posts" FROM "posts" AS "sq1" WHERE (("sq1"."name" LIKE $1) AND sq0.id = sq1.user_id) LIMIT $2) AS "sq1" ON true WHERE exists((SELECT 1 FROM "posts" AS "sq2" INNER JOIN "app"."users" AS "sq3" ON sq0.id = sq2.user_id WHERE exists((SELECT 1 FROM "categories" AS "sq4" INNER JOIN "posts_to_categories" AS "sq5" ON (sq2.id = sq5.post_id AND sq5.category_id = sq4.id) WHERE ("sq4"."name" = $3))))) LIMIT $4`,
			ExpectedArguments: []interface{}{int64(5), int64(100), "%po%", "IT"},
		},
		{
			Name:       "field_name_different_than_table_name",
			SchemaFile: "testdata/schema_simple.graphql",
			GraphQLQuery: `query {
							  users(limit: 5) {
								name
								someOtherName {
								  name
								}
							  }
							}`,
			ExpectedSQL:       `SELECT "sq0"."name" AS "name", "sq1"."someOtherName" AS "someOtherName" FROM "app"."users" AS "sq0" LEFT JOIN LATERAL (SELECT COALESCE(jsonb_agg(jsonb_build_object('name', "sq1"."name")), '[]'::jsonb) AS "someOtherName" FROM "posts" AS "sq1" WHERE sq0.id = sq1.user_id LIMIT $1) AS "sq1" ON true LIMIT $2`,
			ExpectedArguments: []interface{}{int64(5), int64(100)},
		},
		{
			Name:       "query_interface",
			SchemaFile: "testdata/schema_simple.graphql",
			GraphQLQuery: `query {
							  animals {
								id
								name
							}
						}`,
			ExpectedSQL:       `SELECT "sq0"."type" AS "type", "sq0"."id" AS "id", "sq0"."name" AS "name" FROM "app"."animals" AS "sq0" LIMIT $1`,
			ExpectedArguments: []interface{}{int64(100)},
		},
		{
			Name:       "query_on_interface",
			SchemaFile: "testdata/schema_simple.graphql",
			GraphQLQuery: `query {
				animals {
				  id
				  name	
				  ... on Dog {	
					breed	
				  }
				}
			}`,
			ExpectedSQL:       `SELECT "sq0"."type" AS "type", "sq0"."id" AS "id", "sq0"."name" AS "name", "sq0"."breed" AS "breed" FROM "app"."animals" AS "sq0" LIMIT $1`,
			ExpectedArguments: []interface{}{int64(100)},
		},
	}
	_ = os.Chdir("/testdata")
	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			builderTester(t, testCase, func(b sql.Builder, f builders.Field) (string, []interface{}, error) {
				return b.Query(f)
			})
		})

	}

}

func TestBuilder_Insert(t *testing.T) {
	testCases := []TestBuilderCase{
		{
			Name:              "simple_insert",
			SchemaFile:        "testdata/schema_simple.graphql",
			GraphQLQuery:      `mutation { createPosts(inputs: {name: "Ron", id: 111}) { rows_affected posts { name id } } }`,
			ExpectedSQL:       `WITH create_posts AS (INSERT INTO "posts" AS "sq0" ("id", "name") VALUES (111, 'Ron') RETURNING *) SELECT (SELECT COALESCE(jsonb_agg(jsonb_build_object('name', "sq1"."name", 'id', "sq1"."id")), '[]'::jsonb) AS "posts" FROM "create_posts" AS "sq1") AS "posts", (SELECT COUNT(*) AS "rows_affected" FROM "create_posts")`,
			ExpectedArguments: []interface{}{},
		},
		{
			Name:              "multi_insert_query",
			SchemaFile:        "testdata/schema_simple.graphql",
			GraphQLQuery:      `mutation { createPosts(inputs: [{name: "Ron", id: 111}, {name: "Ron", id: 133}]) { rows_affected posts { name id } } }`,
			ExpectedSQL:       `WITH create_posts AS (INSERT INTO "posts" AS "sq0" ("id", "name") VALUES (111, 'Ron'), (133, 'Ron') RETURNING *) SELECT (SELECT COALESCE(jsonb_agg(jsonb_build_object('name', "sq1"."name", 'id', "sq1"."id")), '[]'::jsonb) AS "posts" FROM "create_posts" AS "sq1") AS "posts", (SELECT COUNT(*) AS "rows_affected" FROM "create_posts")`,
			ExpectedArguments: []interface{}{},
		},
	}
	_ = os.Chdir("/testdata")
	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			builderTester(t, testCase, func(b sql.Builder, f builders.Field) (string, []interface{}, error) {
				return b.Create(f)
			})
		})

	}

}

func TestBuilder_Delete(t *testing.T) {
	testCases := []TestBuilderCase{
		{
			Name:              "simple_delete",
			SchemaFile:        "testdata/schema_simple.graphql",
			GraphQLQuery:      `mutation { deletePosts	 { rows_affected posts { name id } } }`,
			ExpectedSQL:       `WITH delete_posts AS (DELETE FROM "posts" RETURNING *) SELECT (SELECT COALESCE(jsonb_agg(jsonb_build_object('name', "sq0"."name", 'id', "sq0"."id")), '[]'::jsonb) AS "posts" FROM "delete_posts" AS "sq0") AS "posts", (SELECT COUNT(*) AS "rows_affected" FROM "delete_posts")`,
			ExpectedArguments: []interface{}{},
		},
		{
			Name:              "delete_with_filter",
			SchemaFile:        "testdata/schema_simple.graphql",
			GraphQLQuery:      "mutation{deletePosts(filter: {id: {eq: 1}}) {rows_affected  posts {name id}}}",
			ExpectedSQL:       `WITH delete_posts AS (DELETE FROM "posts" WHERE ("posts"."id" = 1) RETURNING *) SELECT (SELECT COALESCE(jsonb_agg(jsonb_build_object('name', "sq0"."name", 'id', "sq0"."id")), '[]'::jsonb) AS "posts" FROM "delete_posts" AS "sq0") AS "posts", (SELECT COUNT(*) AS "rows_affected" FROM "delete_posts")`,
			ExpectedArguments: []interface{}{},
		},
	}
	_ = os.Chdir("/testdata")
	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			builderTester(t, testCase, func(b sql.Builder, f builders.Field) (string, []interface{}, error) {
				return b.Delete(f)
			})
		})

	}

}

func TestBuilder_Update(t *testing.T) {
	testCases := []TestBuilderCase{
		{
			Name:              "simple_update",
			SchemaFile:        "testdata/schema_simple.graphql",
			GraphQLQuery:      `mutation { updatePosts(input: {name: "newPost"}) { rows_affected posts { name id } } }`,
			ExpectedSQL:       `WITH update_posts AS (UPDATE "posts" AS "sq0" SET "name"='newPost' RETURNING *) SELECT (SELECT COALESCE(jsonb_agg(jsonb_build_object('name', "sq1"."name", 'id', "sq1"."id")), '[]'::jsonb) AS "posts" FROM "update_posts" AS "sq1") AS "posts", (SELECT COUNT(*) AS "rows_affected" FROM "update_posts")`,
			ExpectedArguments: []interface{}{},
		},
		{
			Name:              "update_with_filter",
			SchemaFile:        "testdata/schema_simple.graphql",
			GraphQLQuery:      `mutation { updatePosts(input: {name: "newPost"}, filter: {id: {eq: 1}}) { rows_affected posts { name id } } }`,
			ExpectedSQL:       `WITH update_posts AS (UPDATE "posts" AS "sq0" SET "name"='newPost' WHERE ("sq0"."id" = 1) RETURNING *) SELECT (SELECT COALESCE(jsonb_agg(jsonb_build_object('name', "sq1"."name", 'id', "sq1"."id")), '[]'::jsonb) AS "posts" FROM "update_posts" AS "sq1") AS "posts", (SELECT COUNT(*) AS "rows_affected" FROM "update_posts")`,
			ExpectedArguments: []interface{}{},
		},
	}
	_ = os.Chdir("/testdata")
	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			builderTester(t, testCase, func(b sql.Builder, f builders.Field) (string, []interface{}, error) {
				return b.Update(f)
			})
		})

	}

}

func TestBuilder_CustomOperator(t *testing.T) {
	testCases := []TestBuilderCase{
		{
			Name:              "base_query",
			SchemaFile:        "testdata/schema_simple.graphql",
			GraphQLQuery:      `query { users { name } }`,
			ExpectedSQL:       `SELECT "sq0"."name" AS "name" FROM "app"."users" AS "sq0" LIMIT $1`,
			ExpectedArguments: []interface{}{int64(100)},
			CustomOperators: map[string]builders.Operator{
				"myCustomOperator": func(table exp.AliasedExpression, key string, value interface{}) goqu.Expression {
					return goqu.L("1 = 1")
				},
			},
		},
		{
			Name:              "query_with_custom_operator",
			SchemaFile:        "testdata/schema_simple.graphql",
			GraphQLQuery:      `query { users(filter: {name: {myCustomOperator: "Test"}}) { name } }`,
			ExpectedSQL:       `SELECT "sq0"."name" AS "name" FROM "app"."users" AS "sq0" WHERE 1 = 1 LIMIT $1`,
			ExpectedArguments: []interface{}{int64(100)},
			CustomOperators: map[string]builders.Operator{
				"myCustomOperator": func(table exp.AliasedExpression, key string, value interface{}) goqu.Expression {
					return goqu.L("1 = 1")
				},
			},
		},
		{
			Name:              "query_with_custom_operator_and_other_filter",
			SchemaFile:        "testdata/schema_simple.graphql",
			GraphQLQuery:      `query { users(filter: {name: {myCustomOperator: "6"}}) { name } }`,
			ExpectedSQL:       `SELECT "sq0"."name" AS "name" FROM "app"."users" AS "sq0" WHERE ("sq0"."name" BETWEEN $1 AND $2) LIMIT $3`,
			ExpectedArguments: []interface{}{int64(1), int64(6), int64(100)},
			CustomOperators: map[string]builders.Operator{
				"myCustomOperator": func(table exp.AliasedExpression, key string, value interface{}) goqu.Expression {
					return table.Col(key).Between(exp.NewRangeVal(1, cast.ToInt(value)))
				},
			},
		},
	}
	_ = os.Chdir("/testdata")
	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			builderTester(t, testCase, func(b sql.Builder, f builders.Field) (string, []interface{}, error) {
				return b.Query(f)
			})
		})

	}
}

func TestBuilder_Query_EdgeCases(t *testing.T) {
	testCases := []TestBuilderCase{
		{
			Name:              "filter_isNull_true",
			SchemaFile:        "testdata/schema_simple.graphql",
			GraphQLQuery:      `query { users(filter: {name: {isNull: true}}) { name } }`,
			ExpectedSQL:       `SELECT "sq0"."name" AS "name" FROM "app"."users" AS "sq0" WHERE ("sq0"."name" IS NULL) LIMIT $1`,
			ExpectedArguments: []interface{}{int64(100)},
		},
		{
			Name:              "filter_isNull_false",
			SchemaFile:        "testdata/schema_simple.graphql",
			GraphQLQuery:      `query { users(filter: {name: {isNull: false}}) { name } }`,
			ExpectedSQL:       `SELECT "sq0"."name" AS "name" FROM "app"."users" AS "sq0" WHERE ("sq0"."name" IS NOT NULL) LIMIT $1`,
			ExpectedArguments: []interface{}{int64(100)},
		},
		{
			Name:              "filter_eq_operator",
			SchemaFile:        "testdata/schema_simple.graphql",
			GraphQLQuery:      `query { users(filter: {name: {eq: "Alice"}}) { name } }`,
			ExpectedSQL:       `SELECT "sq0"."name" AS "name" FROM "app"."users" AS "sq0" WHERE ("sq0"."name" = $1) LIMIT $2`,
			ExpectedArguments: []interface{}{"Alice", int64(100)},
		},
		{
			Name:              "filter_neq_operator",
			SchemaFile:        "testdata/schema_simple.graphql",
			GraphQLQuery:      `query { users(filter: {name: {neq: "Charlie"}}) { name } }`,
			ExpectedSQL:       `SELECT "sq0"."name" AS "name" FROM "app"."users" AS "sq0" WHERE ("sq0"."name" != $1) LIMIT $2`,
			ExpectedArguments: []interface{}{"Charlie", int64(100)},
		},
		{
			Name:              "filter_prefix_operator",
			SchemaFile:        "testdata/schema_simple.graphql",
			GraphQLQuery:      `query { users(filter: {name: {prefix: "Dr."}}) { name } }`,
			ExpectedSQL:       `SELECT "sq0"."name" AS "name" FROM "app"."users" AS "sq0" WHERE ("sq0"."name" LIKE $1) LIMIT $2`,
			ExpectedArguments: []interface{}{"Dr.%", int64(100)},
		},
		{
			Name:              "filter_suffix_operator",
			SchemaFile:        "testdata/schema_simple.graphql",
			GraphQLQuery:      `query { users(filter: {name: {suffix: "son"}}) { name } }`,
			ExpectedSQL:       `SELECT "sq0"."name" AS "name" FROM "app"."users" AS "sq0" WHERE ("sq0"."name" LIKE $1) LIMIT $2`,
			ExpectedArguments: []interface{}{"%son", int64(100)},
		},
		{
			Name:              "ordering_desc",
			SchemaFile:        "testdata/schema_simple.graphql",
			GraphQLQuery:      `query { users(orderBy: {name: DESC}) { name } }`,
			ExpectedSQL:       `SELECT "sq0"."name" AS "name" FROM "app"."users" AS "sq0" ORDER BY "name" DESC NULLS LAST LIMIT $1`,
			ExpectedArguments: []interface{}{int64(100)},
		},
		{
			Name:              "ordering_asc_null_first",
			SchemaFile:        "testdata/schema_simple.graphql",
			GraphQLQuery:      `query { users(orderBy: {name: ASC_NULL_FIRST}) { name } }`,
			ExpectedSQL:       `SELECT "sq0"."name" AS "name" FROM "app"."users" AS "sq0" ORDER BY "name" ASC NULLS FIRST LIMIT $1`,
			ExpectedArguments: []interface{}{int64(100)},
		},
		{
			Name:              "pagination_offset_only",
			SchemaFile:        "testdata/schema_simple.graphql",
			GraphQLQuery:      `query { users(offset: 10) { name } }`,
			ExpectedSQL:       `SELECT "sq0"."name" AS "name" FROM "app"."users" AS "sq0" LIMIT $1 OFFSET $2`,
			ExpectedArguments: []interface{}{int64(100), int64(10)},
		},
		{
			Name:              "pagination_limit_and_offset",
			SchemaFile:        "testdata/schema_simple.graphql",
			GraphQLQuery:      `query { users(limit: 5, offset: 10) { name } }`,
			ExpectedSQL:       `SELECT "sq0"."name" AS "name" FROM "app"."users" AS "sq0" LIMIT $1 OFFSET $2`,
			ExpectedArguments: []interface{}{int64(5), int64(10)},
		},
		{
			Name:              "multiple_filters_combined",
			SchemaFile:        "testdata/schema_simple.graphql",
			GraphQLQuery:      `query { users(filter: {name: {like: "%test%"}, id: {gt: 5}}) { name id } }`,
			ExpectedSQL:       `SELECT "sq0"."name" AS "name", "sq0"."id" AS "id" FROM "app"."users" AS "sq0" WHERE (("sq0"."name" LIKE $1) AND ("sq0"."id" > $2)) LIMIT $3`,
			ExpectedArguments: []interface{}{"%test%", int64(5), int64(100)},
		},
	}
	_ = os.Chdir("/testdata")
	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			builderTester(t, testCase, func(b sql.Builder, f builders.Field) (string, []interface{}, error) {
				return b.Query(f)
			})
		})
	}
}

func TestNewBuilder(t *testing.T) {
	t.Run("with_defaults", func(t *testing.T) {
		schema := &ast.Schema{}
		builder := sql.NewBuilder(&builders.Config{Schema: schema})

		// Verify defaults
		assert.NotNil(t, builder)
		assert.Equal(t, "postgres", builder.Dialect)
		assert.NotNil(t, builder.CaseConverter)
		assert.NotNil(t, builder.Operators)
	})

	t.Run("with_custom_dialect", func(t *testing.T) {
		builder := sql.NewBuilder(&builders.Config{
			Schema:  &ast.Schema{},
			Dialect: "mysql",
		})
		assert.Equal(t, "mysql", builder.Dialect)
	})

	t.Run("with_custom_operators", func(t *testing.T) {
		customOp := func(table exp.AliasedExpression, key string, value interface{}) goqu.Expression {
			return goqu.L("custom")
		}
		builder := sql.NewBuilder(&builders.Config{
			Schema:          &ast.Schema{},
			CustomOperators: map[string]builders.Operator{"custom": customOp},
		})
		_, ok := builder.Operators["custom"]
		assert.True(t, ok)
	})
}

func TestBuilder_Capabilities(t *testing.T) {
	builder := sql.NewBuilder(&builders.Config{Schema: &ast.Schema{}})
	caps := builder.Capabilities()

	assert.True(t, caps.SupportsJoins)
	assert.True(t, caps.SupportsReturning)
	assert.True(t, caps.SupportsTransactions)
	assert.Equal(t, -1, caps.MaxRelationDepth)
}

func builderTester(t *testing.T, testCase TestBuilderCase, caller func(b sql.Builder, f builders.Field) (string, []interface{}, error)) {
	fs := afero.NewOsFs()
	data, err := afero.ReadFile(fs, testCase.SchemaFile)
	require.Nil(t, err)
	testSchema, err := gqlparser.LoadSchema(&ast.Source{
		Name:    "schema.graphql",
		Input:   string(data),
		BuiltIn: false,
	})
	require.Nil(t, err)
	fgqlPlugin := schema.FastGqlPlugin{}
	src, err := fgqlPlugin.CreateAugmented(testSchema)
	require.Nil(t, err)
	augmentedSchema, err := gqlparser.LoadSchema(src...)
	require.Nil(t, err)

	builder := sql.NewBuilder(&builders.Config{
		Schema:             augmentedSchema,
		Logger:             nil,
		TableNameGenerator: &TestTableNameGenerator{},
		CustomOperators:    testCase.CustomOperators,
	})
	doc, err := parser.ParseQuery(&ast.Source{Input: testCase.GraphQLQuery})
	require.Nil(t, err)
	errs := validator.ValidateWithRules(augmentedSchema, doc, nil)
	require.Nil(t, errs)
	def := doc.Operations.ForName("")
	sel := def.SelectionSet[0].(*ast.Field)
	opCtx := &graphql.OperationContext{
		RawQuery:      testCase.GraphQLQuery,
		Variables:     make(map[string]interface{}),
		OperationName: "",
		Doc:           doc,
		Stats:         graphql.Stats{},
	}
	field := builders.CollectFromQuery(sel, augmentedSchema, opCtx, sel.ArgumentMap(nil))
	query, args, err := caller(builder, field)
	assert.Nil(t, err)
	if testCase.ExpectedArguments == nil {
		assert.Len(t, args, 0)
	} else {
		assert.ElementsMatch(t, testCase.ExpectedArguments, args)
	}
	assert.Equal(t, testCase.ExpectedSQL, query)
}
