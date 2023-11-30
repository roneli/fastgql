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

func Test_AugmentOrdering(t *testing.T) {
	// read test file
	testFile, err := os.ReadFile("testdata/ordering_augmentation_test.graphql")
	require.NoError(t, err)
	expectedFile, err := os.ReadFile("testdata/ordering_augmentation_test_expected.graphql")
	require.NoError(t, err)
	expectedFastGqlFile, err := os.ReadFile("testdata/ordering_augmentation_test_fastgql_expected.graphql")
	require.NoError(t, err)

	cfg := config.DefaultConfig()
	cfg.Sources = append([]*ast.Source{{
		Name:    "ordering_augmentation_test.graphql",
		Input:   string(testFile),
		BuiltIn: false,
	}}, &ast.Source{
		Name:    "fastgql.graphql",
		Input:   fastgql,
		BuiltIn: false,
	})
	assert.Nil(t, cfg.LoadSchema())
	sources, err := NewFastGQLPlugin("").CreateAugmented(cfg.Schema, OrderByAugmenter)
	assert.Nil(t, err)
	for _, s := range sources {
		if s.Name == "ordering_augmentation_test.graphql" {
			assert.Equal(t, strings.ReplaceAll(strings.ReplaceAll(string(expectedFile), "\r\n", ""), " ", ""), strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(s.Input, "\n", ""), " ", ""), "\t", ""))
		}
		if s.Name == "fastgql_schema.graphql" {
			assert.Equal(t, strings.ReplaceAll(strings.ReplaceAll(string(expectedFastGqlFile), "\r\n", ""), " ", ""), strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(s.Input, "\n", ""), " ", ""), "\t", ""))
		}
	}
}
