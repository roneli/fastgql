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
