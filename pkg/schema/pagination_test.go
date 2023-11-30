package schema

import (
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
	"os"
	"strings"
	"testing"
)

func Test_AugmentPagination(t *testing.T) {
	// read test file
	testFile, err := os.ReadFile("testdata/pagination_test.graphql")
	require.NoError(t, err)
	expectedFile, err := os.ReadFile("testdata/pagination_test_expected.graphql")
	require.NoError(t, err)

	cfg := config.DefaultConfig()
	cfg.Sources = append([]*ast.Source{{
		Name:    "pagination_test.graphql",
		Input:   string(testFile),
		BuiltIn: false,
	}}, &ast.Source{
		Name:    "fastgql.graphql",
		Input:   fastgql,
		BuiltIn: false,
	})
	assert.Nil(t, cfg.LoadSchema())
	sources, err := NewFastGQLPlugin("test/pagination").CreateAugmented(cfg.Schema, PaginationAugmenter)
	assert.Nil(t, err)
	for _, s := range sources {
		if s.Name == "pagination_test.graphql" {
			// TODO: change to regex
			assert.Equal(t, strings.ReplaceAll(strings.ReplaceAll(string(expectedFile), "\r\n", ""), " ", ""), strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(s.Input, "\n", ""), " ", ""), "\t", ""))
		}
	}
}
