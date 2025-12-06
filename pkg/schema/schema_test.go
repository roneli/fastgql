package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test_GetTableDirective tests table directive parsing
func Test_GetTableDirective(t *testing.T) {
	tests := []struct {
		name             string
		schemaDefinition string
		typeName         string
		expectedTable    *TableDirective
		expectError      bool
	}{
		{
			name: "parses_table_directive_with_all_fields",
			schemaDefinition: `
				type User @table(name: "users", schema: "public", dialect: "postgres") {
					id: ID!
					name: String!
				}
			`,
			typeName: "User",
			expectedTable: &TableDirective{
				Name:    "users",
				Schema:  "public",
				Dialect: "postgres",
			},
			expectError: false,
		},
		{
			name: "parses_table_directive_with_name_only",
			schemaDefinition: `
				type Post @table(name: "posts") {
					id: ID!
				}
			`,
			typeName: "Post",
			expectedTable: &TableDirective{
				Name:    "posts",
				Schema:  "",
				Dialect: "",
			},
			expectError: false,
		},
		{
			name: "parses_table_directive_with_schema",
			schemaDefinition: `
				type Product @table(name: "products", schema: "shop") {
					id: ID!
				}
			`,
			typeName: "Product",
			expectedTable: &TableDirective{
				Name:    "products",
				Schema:  "shop",
				Dialect: "",
			},
			expectError: false,
		},
		{
			name: "returns_error_when_no_directive",
			schemaDefinition: `
				type User {
					id: ID!
					name: String!
				}
			`,
			typeName:    "User",
			expectError: true,
		},
		{
			name: "parses_table_with_dialect",
			schemaDefinition: `
				type Order @table(name: "orders", dialect: "mysql") {
					id: ID!
				}
			`,
			typeName: "Order",
			expectedTable: &TableDirective{
				Name:    "orders",
				Schema:  "",
				Dialect: "mysql",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := buildTestSchema(t, tt.schemaDefinition)
			typeDef := schema.Types[tt.typeName]
			require.NotNil(t, typeDef, "Type should exist")

			result, err := GetTableDirective(typeDef)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tt.expectedTable.Name, result.Name)
				assert.Equal(t, tt.expectedTable.Schema, result.Schema)
				assert.Equal(t, tt.expectedTable.Dialect, result.Dialect)
			}
		})
	}
}

// Test_GetRelationDirective tests relation directive parsing
func Test_GetRelationDirective(t *testing.T) {
	tests := []struct {
		name              string
		schemaDefinition  string
		typeName          string
		fieldName         string
		expectedRelation  *RelationDirective
	}{
		{
			name: "parses_one_to_many_relation",
			schemaDefinition: `
				type User {
					id: ID!
					posts: [Post] @relation(type: ONE_TO_MANY, fields: ["id"], references: ["user_id"])
				}
				type Post {
					id: ID!
				}
			`,
			typeName:  "User",
			fieldName: "posts",
			expectedRelation: &RelationDirective{
				RelType:    OneToMany,
				Fields:     []string{"id"},
				References: []string{"user_id"},
			},
		},
		{
			name: "parses_one_to_one_relation",
			schemaDefinition: `
				type User {
					id: ID!
					profile: Profile @relation(type: ONE_TO_ONE, fields: ["id"], references: ["user_id"])
				}
				type Profile {
					id: ID!
				}
			`,
			typeName:  "User",
			fieldName: "profile",
			expectedRelation: &RelationDirective{
				RelType:    OneToOne,
				Fields:     []string{"id"},
				References: []string{"user_id"},
			},
		},
		{
			name: "parses_many_to_many_relation",
			schemaDefinition: `
				type Post {
					id: ID!
					categories: [Category] @relation(
						type: MANY_TO_MANY,
						fields: ["id"],
						references: ["id"],
						manyToManyTable: "posts_to_categories",
						manyToManyFields: ["post_id"],
						manyToManyReferences: ["category_id"]
					)
				}
				type Category {
					id: ID!
				}
			`,
			typeName:  "Post",
			fieldName: "categories",
			expectedRelation: &RelationDirective{
				RelType:              ManyToMany,
				Fields:               []string{"id"},
				References:           []string{"id"},
				ManyToManyTable:      "posts_to_categories",
				ManyToManyFields:     []string{"post_id"},
				ManyToManyReferences: []string{"category_id"},
			},
		},
		{
			name: "returns_nil_when_no_directive",
			schemaDefinition: `
				type User {
					id: ID!
					posts: [Post]
				}
				type Post {
					id: ID!
				}
			`,
			typeName:         "User",
			fieldName:        "posts",
			expectedRelation: nil,
		},
		{
			name: "parses_relation_with_multiple_fields",
			schemaDefinition: `
				type User {
					id: ID!
					posts: [Post] @relation(
						type: ONE_TO_MANY,
						fields: ["id", "tenant_id"],
						references: ["user_id", "tenant_id"]
					)
				}
				type Post {
					id: ID!
				}
			`,
			typeName:  "User",
			fieldName: "posts",
			expectedRelation: &RelationDirective{
				RelType:    OneToMany,
				Fields:     []string{"id", "tenant_id"},
				References: []string{"user_id", "tenant_id"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := buildTestSchema(t, tt.schemaDefinition)
			typeDef := schema.Types[tt.typeName]
			require.NotNil(t, typeDef, "Type should exist")

			field := typeDef.Fields.ForName(tt.fieldName)
			require.NotNil(t, field, "Field should exist")

			result := GetRelationDirective(field)

			if tt.expectedRelation == nil {
				assert.Nil(t, result, "Should return nil when no directive")
			} else {
				require.NotNil(t, result, "Should return relation directive")
				assert.Equal(t, tt.expectedRelation.RelType, result.RelType)
				assert.Equal(t, tt.expectedRelation.Fields, result.Fields)
				assert.Equal(t, tt.expectedRelation.References, result.References)
				
				if tt.expectedRelation.ManyToManyTable != "" {
					assert.Equal(t, tt.expectedRelation.ManyToManyTable, result.ManyToManyTable)
					assert.Equal(t, tt.expectedRelation.ManyToManyFields, result.ManyToManyFields)
					assert.Equal(t, tt.expectedRelation.ManyToManyReferences, result.ManyToManyReferences)
				}
			}
		})
	}
}

// Test_getArgumentValue tests argument value extraction
func Test_getArgumentValue(t *testing.T) {
	tests := []struct {
		name             string
		schemaDefinition string
		typeName         string
		directiveName    string
		argName          string
		expected         string
	}{
		{
			name: "returns_string_value",
			schemaDefinition: `
				type User @table(name: "users") {
					id: ID!
				}
			`,
			typeName:      "User",
			directiveName: "table",
			argName:       "name",
			expected:      "users",
		},
		{
			name: "returns_empty_string_when_arg_not_found",
			schemaDefinition: `
				type User @table(name: "users") {
					id: ID!
				}
			`,
			typeName:      "User",
			directiveName: "table",
			argName:       "nonexistent",
			expected:      "",
		},
		{
			name: "returns_value_from_multiple_args",
			schemaDefinition: `
				type User @table(name: "users", schema: "public", dialect: "postgres") {
					id: ID!
				}
			`,
			typeName:      "User",
			directiveName: "table",
			argName:       "schema",
			expected:      "public",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := buildTestSchema(t, tt.schemaDefinition)
			typeDef := schema.Types[tt.typeName]
			require.NotNil(t, typeDef)

			directive := typeDef.Directives.ForName(tt.directiveName)
			require.NotNil(t, directive)

			result := getArgumentValue(directive.Arguments, tt.argName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

