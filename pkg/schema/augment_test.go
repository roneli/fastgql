package schema

import (
	_ "embed"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
)

//go:embed fastgql.graphql
var fastgql string

type generateTestCase struct {
	name                      string
	baseSchemaFile            string
	expectedSchemaFile        string
	expectedFastgqlSchemaFile string
	Augmenter                 []Augmenter
}

func generateTestRunner(t *testing.T, tc *generateTestCase, augmenters ...Augmenter) {
	// read test file
	testFile, err := os.ReadFile(tc.baseSchemaFile)
	require.NoError(t, err)
	expectedSchemaFile, err := os.ReadFile(tc.expectedSchemaFile)
	require.NoError(t, err)
	var expectedFastgqlSchemaFile []byte
	if tc.expectedFastgqlSchemaFile != "" {
		expectedFastgqlSchemaFile, err = os.ReadFile(tc.expectedFastgqlSchemaFile)
		require.NoError(t, err)
	}

	cfg := config.DefaultConfig()
	cfg.Sources = append([]*ast.Source{{
		Name:    "tc.graphql",
		Input:   string(testFile),
		BuiltIn: false,
	}}, &ast.Source{
		Name:    "fastgql.graphql",
		Input:   fastgql,
		BuiltIn: false,
	})
	assert.Nil(t, cfg.LoadSchema())
	sources, err := NewFastGQLPlugin("").CreateAugmented(cfg.Schema, append(augmenters, tc.Augmenter...)...)
	assert.Nil(t, err)
	for _, s := range sources {
		switch s.Name {
		case "tc.graphql":
			if !assert.Equal(t, strings.ReplaceAll(strings.ReplaceAll(string(expectedSchemaFile), "\r\n", ""), " ", ""), strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(s.Input, "\n", ""), " ", ""), "\t", "")) {
				fmt.Print(s.Input)
			}
		case "fastgql_schema.graphql":
			if !assert.Equal(t, strings.ReplaceAll(strings.ReplaceAll(string(expectedFastgqlSchemaFile), "\r\n", ""), " ", ""), strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(s.Input, "\n", ""), " ", ""), "\t", "")) {
				fmt.Print(s.Input)
			}
		}
	}
}
