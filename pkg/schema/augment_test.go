package schema

import (
	_ "embed"
	"fmt"
	"os"
	"regexp"
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
	sources, err := NewFastGQLPlugin("", "", false).CreateAugmented(cfg.Schema, append(augmenters, tc.Augmenter...)...)
	assert.Nil(t, err)
	for _, s := range sources {
		switch s.Name {
		case "tc.graphql":
			if !assert.Equal(t, removeWhitespaceWithRegex(string(expectedSchemaFile)), removeWhitespaceWithRegex(s.Input)) {
				fmt.Print(s.Input)
			}
		case "fastgql_schema.graphql":
			if !assert.Equal(t, removeWhitespaceWithRegex(string(expectedFastgqlSchemaFile)), removeWhitespaceWithRegex(s.Input)) {
				fmt.Print(s.Input)
			}
		}
	}
}

func removeWhitespaceWithRegex(s string) string {
	reg := regexp.MustCompile(`[\s]+`) // Match any whitespace character
	return reg.ReplaceAllString(s, "")
}

// Test_skipAugment tests the skipAugment helper function
func Test_skipAugment(t *testing.T) {
	tests := []struct {
		name             string
		schemaDefinition string
		typeName         string
		fieldName        string
		checkArgs        []string
		shouldSkip       bool
	}{
		{
			name: "skips_field_when_arg_exists",
			schemaDefinition: `
				type User {
					id: ID!
				}
				type Query {
					users(filter: String): [User]
				}
			`,
			typeName:   "Query",
			fieldName:  "users",
			checkArgs:  []string{"filter"},
			shouldSkip: true,
		},
		{
			name: "does_not_skip_normal_field",
			schemaDefinition: `
				type User {
					id: ID!
					name: String
				}
			`,
			typeName:   "User",
			fieldName:  "name",
			checkArgs:  []string{},
			shouldSkip: false,
		},
		{
			name: "does_not_skip_when_different_arg",
			schemaDefinition: `
				type User {
					id: ID!
				}
				type Query {
					users(limit: Int): [User]
				}
			`,
			typeName:   "Query",
			fieldName:  "users",
			checkArgs:  []string{"filter"},
			shouldSkip: false,
		},
		{
			name: "skips_when_any_of_multiple_args_exist",
			schemaDefinition: `
				type User {
					id: ID!
				}
				type Query {
					users(limit: Int, offset: Int): [User]
				}
			`,
			typeName:   "Query",
			fieldName:  "users",
			checkArgs:  []string{"limit", "orderBy"},
			shouldSkip: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := buildTestSchema(t, tt.schemaDefinition)
			typeDef := schema.Types[tt.typeName]
			require.NotNil(t, typeDef, "Type should exist")

			field := typeDef.Fields.ForName(tt.fieldName)
			require.NotNil(t, field, "Field should exist")

			result := skipAugment(field, tt.checkArgs...)
			assert.Equal(t, tt.shouldSkip, result)
		})
	}
}

// Test_addRecursive tests recursive augmentation
func Test_addRecursive(t *testing.T) {
	tests := []struct {
		name             string
		schemaDefinition string
		typeName         string
		fieldStopCase    string
		expectedCalls    int
		expectError      bool
	}{
		{
			name: "recurses_into_list_fields",
			schemaDefinition: `
				type User {
					id: ID!
					posts: [Post]
				}
				type Post {
					id: ID!
					comments: [Comment]
				}
				type Comment {
					id: ID!
				}
			`,
			typeName:      "User",
			fieldStopCase: "test",
			expectedCalls: 2, // posts and comments
			expectError:   false,
		},
		{
			name: "skips_non_list_fields",
			schemaDefinition: `
				type User {
					id: ID!
					profile: Profile
				}
				type Profile {
					bio: String
				}
			`,
			typeName:      "User",
			fieldStopCase: "test",
			expectedCalls: 0,
			expectError:   false,
		},
		{
			name: "skips_fields_with_stopCase_arg",
			schemaDefinition: `
				type User {
					id: ID!
					posts(filter: String): [Post]
				}
				type Post {
					id: ID!
				}
			`,
			typeName:      "User",
			fieldStopCase: "filter",
			expectedCalls: 0,
			expectError:   false,
		},
		{
			name: "handles_circular_references",
			schemaDefinition: `
				type User {
					id: ID!
					friends: [User]
				}
			`,
			typeName:      "User",
			fieldStopCase: "test",
			expectedCalls: 1, // friends (only once due to visited tracking)
			expectError:   false,
		},
		{
			name: "skips_scalar_list_fields",
			schemaDefinition: `
				type User {
					id: ID!
					tags: [String]
				}
			`,
			typeName:      "User",
			fieldStopCase: "test",
			expectedCalls: 0, // Scalar lists are skipped
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := buildTestSchema(t, tt.schemaDefinition)
			typeDef := schema.Types[tt.typeName]
			require.NotNil(t, typeDef, "Type should exist")

			callCount := 0
			mockAugmenter := func(s *ast.Schema, obj *ast.Definition, field *ast.FieldDefinition) error {
				callCount++
				return nil
			}

			err := addRecursive(schema, typeDef, tt.fieldStopCase, mockAugmenter)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedCalls, callCount, "Expected %d augmenter calls, got %d", tt.expectedCalls, callCount)
			}
		})
	}
}

// Test_hasVisited tests the visited tracking helper
func Test_hasVisited(t *testing.T) {
	// Create definitions for testing
	userDef := &ast.Definition{Name: "User"}
	postDef := &ast.Definition{Name: "Post"}
	commentDef := &ast.Definition{Name: "Comment"}

	tests := []struct {
		name     string
		obj      *ast.Definition
		visited  []*ast.Definition
		expected bool
	}{
		{
			name:     "returns_false_for_empty_visited",
			obj:      userDef,
			visited:  []*ast.Definition{},
			expected: false,
		},
		{
			name:     "returns_true_when_visited",
			obj:      userDef,
			visited:  []*ast.Definition{postDef, userDef, commentDef},
			expected: true,
		},
		{
			name:     "returns_false_when_not_visited",
			obj:      commentDef,
			visited:  []*ast.Definition{userDef, postDef},
			expected: false,
		},
		{
			name:     "uses_pointer_equality",
			obj:      &ast.Definition{Name: "User"},
			visited:  []*ast.Definition{userDef}, // Different pointer
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasVisited(tt.obj, tt.visited)
			assert.Equal(t, tt.expected, result)
		})
	}
}
