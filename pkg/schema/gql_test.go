package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
)

// Test_IsListType tests the list type detection
func Test_IsListType(t *testing.T) {
	tests := []struct {
		name     string
		astType  *ast.Type
		expected bool
	}{
		{
			name: "returns_true_for_list_type",
			astType: &ast.Type{
				Elem: &ast.Type{
					NamedType: "String",
				},
			},
			expected: true,
		},
		{
			name: "returns_false_for_scalar_type",
			astType: &ast.Type{
				NamedType: "String",
			},
			expected: false,
		},
		{
			name: "returns_true_for_nested_list",
			astType: &ast.Type{
				Elem: &ast.Type{
					Elem: &ast.Type{
						NamedType: "Int",
					},
				},
			},
			expected: true,
		},
		{
			name: "returns_false_for_null",
			astType: &ast.Type{
				NamedType: "ID",
				NonNull:   true,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsListType(tt.astType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test_IsScalarListType tests scalar list type detection
func Test_IsScalarListType(t *testing.T) {
	tests := []struct {
		name             string
		schemaDefinition string
		typeName         string
		isListType       bool
		expected         bool
	}{
		{
			name: "returns_true_for_list_of_scalars",
			schemaDefinition: `
				type User {
					tags: [String]
				}
			`,
			typeName:   "tags",
			isListType: true,
			expected:   true,
		},
		{
			name: "returns_false_for_scalar",
			schemaDefinition: `
				type User {
					name: String
				}
			`,
			typeName:   "name",
			isListType: false,
			expected:   false,
		},
		{
			name: "returns_false_for_list_of_objects",
			schemaDefinition: `
				type User {
					posts: [Post]
				}
				type Post {
					id: ID
				}
			`,
			typeName:   "posts",
			isListType: true,
			expected:   false,
		},
		{
			name: "returns_true_for_list_of_enums",
			schemaDefinition: `
				type User {
					roles: [Role]
				}
				enum Role {
					ADMIN
					USER
				}
			`,
			typeName:   "roles",
			isListType: true,
			expected:   true,
		},
		{
			name: "returns_true_for_list_of_ids",
			schemaDefinition: `
				type User {
					ids: [ID]
				}
			`,
			typeName:   "ids",
			isListType: true,
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := buildTestSchema(t, tt.schemaDefinition)
			userType := schema.Types["User"]
			require.NotNil(t, userType)

			field := userType.Fields.ForName(tt.typeName)
			require.NotNil(t, field, "Field should exist")

			result := IsScalarListType(schema, field.Type)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test_GetType tests type unwrapping
func Test_GetType(t *testing.T) {
	tests := []struct {
		name         string
		astType      *ast.Type
		expectedName string
	}{
		{
			name: "returns_type_for_simple_scalar",
			astType: &ast.Type{
				NamedType: "String",
			},
			expectedName: "String",
		},
		{
			name: "unwraps_list_type",
			astType: &ast.Type{
				Elem: &ast.Type{
					NamedType: "Int",
				},
			},
			expectedName: "Int",
		},
		{
			name: "unwraps_nested_list_type",
			astType: &ast.Type{
				Elem: &ast.Type{
					Elem: &ast.Type{
						NamedType: "Float",
					},
				},
			},
			expectedName: "Float",
		},
		{
			name: "unwraps_non_null_list",
			astType: &ast.Type{
				Elem: &ast.Type{
					NamedType: "ID",
					NonNull:   true,
				},
			},
			expectedName: "ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetType(tt.astType)
			assert.Equal(t, tt.expectedName, result.Name())
		})
	}
}

// Test_GetDirectiveValue tests directive value extraction
func Test_GetDirectiveValue(t *testing.T) {
	tests := []struct {
		name             string
		schemaDefinition string
		typeName         string
		directiveName    string
		argName          string
		expected         interface{}
	}{
		{
			name: "returns_string_value",
			schemaDefinition: `
				type User @table(name: "users", schema: "public") {
					id: ID!
				}
			`,
			typeName:      "User",
			directiveName: "table",
			argName:       "name",
			expected:      "users",
		},
		{
			name: "returns_bool_value",
			schemaDefinition: `
				type User {
					id: ID!
				}
				type Query {
					users: [User] @generate(filter: true, pagination: false)
				}
			`,
			typeName:      "Query",
			directiveName: "generate",
			argName:       "filter",
			expected:      true,
		},
		{
			name: "returns_false_bool_value",
			schemaDefinition: `
				type User {
					id: ID!
				}
				type Query {
					users: [User] @generate(pagination: false)
				}
			`,
			typeName:      "Query",
			directiveName: "generate",
			argName:       "pagination",
			expected:      false,
		},
		{
			name: "returns_nil_when_directive_not_found",
			schemaDefinition: `
				type User {
					id: ID!
				}
			`,
			typeName:      "User",
			directiveName: "nonexistent",
			argName:       "name",
			expected:      nil,
		},
		{
			name: "returns_nil_when_arg_not_found",
			schemaDefinition: `
				type User @table(name: "users") {
					id: ID!
				}
			`,
			typeName:      "User",
			directiveName: "table",
			argName:       "nonexistent",
			expected:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := buildTestSchema(t, tt.schemaDefinition)

			var directive *ast.Directive
			if tt.typeName == "Query" {
				queryType := schema.Types["Query"]
				require.NotNil(t, queryType)
				field := queryType.Fields.ForName("users")
				require.NotNil(t, field)
				directive = field.Directives.ForName(tt.directiveName)
			} else {
				typeDef := schema.Types[tt.typeName]
				require.NotNil(t, typeDef)
				directive = typeDef.Directives.ForName(tt.directiveName)
			}

			result := GetDirectiveValue(directive, tt.argName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test_GetDirectiveValue_NilDirective tests nil directive handling
func Test_GetDirectiveValue_NilDirective(t *testing.T) {
	result := GetDirectiveValue(nil, "anyArg")
	assert.Nil(t, result, "Should return nil for nil directive")
}


