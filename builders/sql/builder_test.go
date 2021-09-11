package sql_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/vektah/gqlparser/v2/validator"

	"github.com/vektah/gqlparser/v2/parser"

	"github.com/roneli/fastgql/builders"
	"github.com/roneli/fastgql/builders/sql"
	"github.com/vektah/gqlparser/v2"

	"github.com/spf13/afero"
	"github.com/vektah/gqlparser/v2/ast"

	"github.com/stretchr/testify/assert"

	"github.com/roneli/fastgql/schema"
)

type TestBuilderCase struct {
	Name              string
	SchemaFile        string
	GraphQLQuery      string
	ExpectedArguments []interface{}
	ExpectedSQL       string
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
			ExpectedSQL:       `SELECT "sq0"."name" AS "name" FROM "app"."user" AS "sq0" LIMIT $1`,
			ExpectedArguments: []interface{}{int64(100)},
		},
		{
			Name:              "query_with_limit",
			SchemaFile:        "testdata/schema_simple.graphql",
			GraphQLQuery:      `query { users(limit:5) { name } }`,
			ExpectedSQL:       `SELECT "sq0"."name" AS "name" FROM "app"."user" AS "sq0" LIMIT $1`,
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
			ExpectedSQL:       `SELECT "sq0"."name" AS "name", "sq1"."posts" AS "posts" FROM "app"."user" AS "sq0" LEFT JOIN LATERAL (SELECT COALESCE(jsonb_agg(jsonb_build_object('name', "sq1"."name")), '[]'::jsonb) AS "posts" FROM "posts" AS "sq1" WHERE (("sq1"."name" LIKE $1) AND sq0.id = sq1.user_id) LIMIT $2) AS "sq1" ON true WHERE exists((SELECT 1 FROM "posts" AS "sq2" INNER JOIN "user" AS "sq3" ON sq0.id = sq2.user_id WHERE exists((SELECT 1 FROM "categories" AS "sq4" INNER JOIN "posts_to_categories" AS "sq5" ON (sq2.id = sq5.post_id AND sq5.category_id = sq4.id) WHERE ("sq4"."name" = $3))))) LIMIT $4`,
			ExpectedArguments: []interface{}{int64(5), int64(100), "%po%", "IT"},
		},
	}
	_ = os.Chdir("/testdata")
	fs := afero.NewOsFs()
	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			data, err := afero.ReadFile(fs, testCase.SchemaFile)
			assert.Nil(t, err)
			testSchema, err := gqlparser.LoadSchema(&ast.Source{
				Name:    "schema.graphql",
				Input:   string(data),
				BuiltIn: false,
			})
			assert.Nil(t, err)
			fgqlPlugin := schema.FastGqlPlugin{}
			src := fgqlPlugin.CreateAugmented(testSchema)
			augmentedSchema, err := gqlparser.LoadSchema(src)
			assert.Nil(t, err)

			builder := sql.NewBuilder(&builders.Config{
				Schema:             augmentedSchema,
				Logger:             nil,
				TableNameGenerator: &TestTableNameGenerator{},
			})
			doc, err := parser.ParseQuery(&ast.Source{Input: testCase.GraphQLQuery})
			assert.Nil(t, err)
			errs := validator.Validate(augmentedSchema, doc)
			assert.Nil(t, errs)
			def := doc.Operations.ForName("")
			sel := def.SelectionSet[0].(*ast.Field)
			field := builders.CollectFromQuery(sel, doc, make(map[string]interface{}), sel.ArgumentMap(nil))
			query, args, err := builder.Query(field)
			assert.Nil(t, err)
			if testCase.ExpectedArguments == nil {
				assert.Len(t, args, 0)
			} else {
				assert.ElementsMatch(t, testCase.ExpectedArguments, args)
			}
			assert.Equal(t, testCase.ExpectedSQL, query)
		})

	}

}