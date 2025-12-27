package importer

import (
	"strings"
	"testing"

	schemapkg "github.com/roneli/fastgql/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateSchema(t *testing.T) {
	tests := []struct {
		name     string
		schema   *Schema
		expected string
		wantErr  bool
	}{
		{
			name: "simple_object_with_basic_fields",
			schema: &Schema{
				ObjectTypes: []*ObjectType{
					{
						Name:      "User",
						TableName: "users",
						Dialect:   "postgres",
						Fields: []*Field{
							{Name: "id", Type: schemapkg.GraphQLTypeInt, IsNonNull: true},
							{Name: "name", Type: schemapkg.GraphQLTypeString, IsNonNull: true},
							{Name: "email", Type: schemapkg.GraphQLTypeString, IsNonNull: false},
						},
					},
				},
			},
			expected: `type User @table(name: "users", dialect: "postgres") {
  id: Int!
  name: String!
  email: String
}
`,
		},
		{
			name: "object_with_filter_input",
			schema: &Schema{
				ObjectTypes: []*ObjectType{
					{
						Name:                "Post",
						TableName:           "posts",
						Dialect:             "postgres",
						GenerateFilterInput: true,
						Fields: []*Field{
							{Name: "id", Type: schemapkg.GraphQLTypeInt, IsNonNull: true},
							{Name: "title", Type: schemapkg.GraphQLTypeString, IsNonNull: true},
						},
					},
				},
			},
			expected: `type Post @table(name: "posts", dialect: "postgres") @generateFilterInput {
  id: Int!
  title: String!
}
`,
		},
		{
			name: "object_with_schema_name",
			schema: &Schema{
				ObjectTypes: []*ObjectType{
					{
						Name:      "Product",
						TableName: "products",
						Schema:    "shop",
						Dialect:   "postgres",
						Fields: []*Field{
							{Name: "id", Type: schemapkg.GraphQLTypeInt, IsNonNull: true},
						},
					},
				},
			},
			expected: `type Product @table(name: "products", dialect: "postgres", schema: "shop") {
  id: Int!
}
`,
		},
		{
			name: "object_with_list_fields",
			schema: &Schema{
				ObjectTypes: []*ObjectType{
					{
						Name:      "Tag",
						TableName: "tags",
						Dialect:   "postgres",
						Fields: []*Field{
							{Name: "id", Type: schemapkg.GraphQLTypeInt, IsNonNull: true},
							{Name: "names", Type: schemapkg.GraphQLTypeString, IsList: true, IsNonNull: true},
							{Name: "values", Type: schemapkg.GraphQLTypeInt, IsList: true, IsNonNull: false},
						},
					},
				},
			},
			expected: `type Tag @table(name: "tags", dialect: "postgres") {
  id: Int!
  names: [String]!
  values: [Int]
}
`,
		},
		{
			name: "object_with_json_field",
			schema: &Schema{
				ObjectTypes: []*ObjectType{
					{
						Name:      "Product",
						TableName: "products",
						Dialect:   "postgres",
						Fields: []*Field{
							{Name: "id", Type: schemapkg.GraphQLTypeInt, IsNonNull: true},
							{Name: "attributes", Type: schemapkg.GraphQLTypeMap, IsJSON: true, JSONColumnName: "attributes"},
						},
					},
				},
			},
			expected: `type Product @table(name: "products", dialect: "postgres") {
  id: Int!
  attributes: Map @json(column: "attributes")
}
`,
		},
		{
			name: "object_with_one_to_one_relation",
			schema: &Schema{
				ObjectTypes: []*ObjectType{
					{
						Name:      "User",
						TableName: "users",
						Dialect:   "postgres",
						Fields: []*Field{
							{Name: "id", Type: schemapkg.GraphQLTypeInt, IsNonNull: true},
						},
					},
					{
						Name:      "Profile",
						TableName: "profiles",
						Dialect:   "postgres",
						Fields: []*Field{
							{Name: "id", Type: schemapkg.GraphQLTypeInt, IsNonNull: true},
							{Name: "userId", Type: schemapkg.GraphQLTypeInt, IsNonNull: true},
						},
						Relations: []*Relation{
							{
								FieldName:  "user",
								Type:       schemapkg.OneToOne,
								Fields:     []string{"userId"},
								References: []string{"id"},
								TargetType: "User",
							},
						},
					},
				},
			},
			expected: `type User @table(name: "users", dialect: "postgres") {
  id: Int!
}

type Profile @table(name: "profiles", dialect: "postgres") {
  id: Int!
  userId: Int!
  user: User @relation(type: ONE_TO_ONE, fields: ["userId"], references: ["id"])
}
`,
		},
		{
			name: "object_with_one_to_many_relation",
			schema: &Schema{
				ObjectTypes: []*ObjectType{
					{
						Name:      "User",
						TableName: "users",
						Dialect:   "postgres",
						Fields: []*Field{
							{Name: "id", Type: schemapkg.GraphQLTypeInt, IsNonNull: true},
						},
						Relations: []*Relation{
							{
								FieldName:  "posts",
								Type:       schemapkg.OneToMany,
								Fields:     []string{"id"},
								References: []string{"userId"},
								TargetType: "Post",
							},
						},
					},
					{
						Name:      "Post",
						TableName: "posts",
						Dialect:   "postgres",
						Fields: []*Field{
							{Name: "id", Type: schemapkg.GraphQLTypeInt, IsNonNull: true},
							{Name: "userId", Type: schemapkg.GraphQLTypeInt, IsNonNull: true},
						},
					},
				},
			},
			expected: `type User @table(name: "users", dialect: "postgres") {
  id: Int!
  posts: [Post] @relation(type: ONE_TO_MANY, fields: ["id"], references: ["userId"])
}

type Post @table(name: "posts", dialect: "postgres") {
  id: Int!
  userId: Int!
}
`,
		},
		{
			name: "object_with_many_to_many_relation",
			schema: &Schema{
				ObjectTypes: []*ObjectType{
					{
						Name:      "Post",
						TableName: "posts",
						Dialect:   "postgres",
						Fields: []*Field{
							{Name: "id", Type: schemapkg.GraphQLTypeInt, IsNonNull: true},
						},
						Relations: []*Relation{
							{
								FieldName:            "categories",
								Type:                 schemapkg.ManyToMany,
								Fields:               []string{"id"},
								References:           []string{"id"},
								TargetType:           "Category",
								ManyToManyTable:      "posts_to_categories",
								ManyToManyFields:     []string{"post_id"},
								ManyToManyReferences: []string{"category_id"},
							},
						},
					},
					{
						Name:      "Category",
						TableName: "categories",
						Dialect:   "postgres",
						Fields: []*Field{
							{Name: "id", Type: schemapkg.GraphQLTypeInt, IsNonNull: true},
						},
					},
				},
			},
			expected: `type Post @table(name: "posts", dialect: "postgres") {
  id: Int!
  categories: [Category] @relation(type: MANY_TO_MANY, fields: ["id"], references: ["id"], manyToManyTable: "posts_to_categories", manyToManyFields: ["post_id"], manyToManyReferences: ["category_id"])
}

type Category @table(name: "categories", dialect: "postgres") {
  id: Int!
}
`,
		},
		{
			name: "schema_with_query_fields",
			schema: &Schema{
				ObjectTypes: []*ObjectType{
					{
						Name:      "User",
						TableName: "users",
						Dialect:   "postgres",
						Fields: []*Field{
							{Name: "id", Type: schemapkg.GraphQLTypeInt, IsNonNull: true},
						},
					},
				},
				QueryFields: []*QueryField{
					{Name: "users", Type: "User", IsList: true, Generate: true},
					{Name: "user", Type: "User", IsList: false, Generate: false},
				},
			},
			expected: `type User @table(name: "users", dialect: "postgres") {
  id: Int!
}

type Query {
  users: [User] @generate
  user: User
}
`,
		},
		{
			name: "complex_schema_with_multiple_types",
			schema: &Schema{
				ObjectTypes: []*ObjectType{
					{
						Name:                "User",
						TableName:           "users",
						Dialect:             "postgres",
						GenerateFilterInput: true,
						Fields: []*Field{
							{Name: "id", Type: schemapkg.GraphQLTypeInt, IsNonNull: true},
							{Name: "name", Type: schemapkg.GraphQLTypeString, IsNonNull: true},
							{Name: "email", Type: schemapkg.GraphQLTypeString, IsNonNull: false},
						},
						Relations: []*Relation{
							{
								FieldName:  "posts",
								Type:       schemapkg.OneToMany,
								Fields:     []string{"id"},
								References: []string{"userId"},
								TargetType: "Post",
							},
						},
					},
					{
						Name:                "Post",
						TableName:           "posts",
						Dialect:             "postgres",
						GenerateFilterInput: true,
						Fields: []*Field{
							{Name: "id", Type: schemapkg.GraphQLTypeInt, IsNonNull: true},
							{Name: "title", Type: schemapkg.GraphQLTypeString, IsNonNull: true},
							{Name: "userId", Type: schemapkg.GraphQLTypeInt, IsNonNull: true},
						},
						Relations: []*Relation{
							{
								FieldName:  "user",
								Type:       schemapkg.OneToOne,
								Fields:     []string{"userId"},
								References: []string{"id"},
								TargetType: "User",
							},
						},
					},
				},
				QueryFields: []*QueryField{
					{Name: "users", Type: "User", IsList: true, Generate: true},
					{Name: "posts", Type: "Post", IsList: true, Generate: true},
				},
			},
			expected: `type User @table(name: "users", dialect: "postgres") @generateFilterInput {
  id: Int!
  name: String!
  email: String
  posts: [Post] @relation(type: ONE_TO_MANY, fields: ["id"], references: ["userId"])
}

type Post @table(name: "posts", dialect: "postgres") @generateFilterInput {
  id: Int!
  title: String!
  userId: Int!
  user: User @relation(type: ONE_TO_ONE, fields: ["userId"], references: ["id"])
}

type Query {
  users: [User] @generate
  posts: [Post] @generate
}
`,
		},
		{
			name:    "nil_schema_returns_error",
			schema:  nil,
			wantErr: true,
		},
		{
			name: "empty_schema_generates_empty_output",
			schema: &Schema{
				ObjectTypes: []*ObjectType{},
			},
			expected: "",
		},
		{
			name: "schema_without_query_fields",
			schema: &Schema{
				ObjectTypes: []*ObjectType{
					{
						Name:      "User",
						TableName: "users",
						Dialect:   "postgres",
						Fields: []*Field{
							{Name: "id", Type: schemapkg.GraphQLTypeInt, IsNonNull: true},
						},
					},
				},
				QueryFields: []*QueryField{},
			},
			expected: `type User @table(name: "users", dialect: "postgres") {
  id: Int!
}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GenerateSchema(tt.schema)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, result)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)

			// Normalize whitespace for comparison
			expected := normalizeWhitespace(tt.expected)
			actual := normalizeWhitespace(result.Input)

			assert.Equal(t, expected, actual, "Generated schema should match expected")
		})
	}
}

func TestGenerateSchema_FieldTypes(t *testing.T) {
	tests := []struct {
		name     string
		field    *Field
		expected string
	}{
		{
			name:     "int_non_null",
			field:    &Field{Name: "id", Type: schemapkg.GraphQLTypeInt, IsNonNull: true},
			expected: "id: Int!",
		},
		{
			name:     "int_nullable",
			field:    &Field{Name: "id", Type: schemapkg.GraphQLTypeInt, IsNonNull: false},
			expected: "id: Int",
		},
		{
			name:     "string_non_null",
			field:    &Field{Name: "name", Type: schemapkg.GraphQLTypeString, IsNonNull: true},
			expected: "name: String!",
		},
		{
			name:     "boolean_non_null",
			field:    &Field{Name: "active", Type: schemapkg.GraphQLTypeBoolean, IsNonNull: true},
			expected: "active: Boolean!",
		},
		{
			name:     "float_non_null",
			field:    &Field{Name: "price", Type: schemapkg.GraphQLTypeFloat, IsNonNull: true},
			expected: "price: Float!",
		},
		{
			name:     "id_non_null",
			field:    &Field{Name: "uuid", Type: schemapkg.GraphQLTypeID, IsNonNull: true},
			expected: "uuid: ID!",
		},
		{
			name:     "list_non_null",
			field:    &Field{Name: "tags", Type: schemapkg.GraphQLTypeString, IsList: true, IsNonNull: true},
			expected: "tags: [String]!",
		},
		{
			name:     "list_nullable",
			field:    &Field{Name: "tags", Type: schemapkg.GraphQLTypeString, IsList: true, IsNonNull: false},
			expected: "tags: [String]",
		},
		{
			name:     "json_field",
			field:    &Field{Name: "metadata", Type: schemapkg.GraphQLTypeMap, IsJSON: true, JSONColumnName: "metadata"},
			expected: `metadata: Map @json(column: "metadata")`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fieldToGraphQL(tt.field)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateSchema_RelationTypes(t *testing.T) {
	tests := []struct {
		name     string
		relation *Relation
		expected string
	}{
		{
			name: "one_to_one",
			relation: &Relation{
				FieldName:  "profile",
				Type:       schemapkg.OneToOne,
				Fields:     []string{"userId"},
				References: []string{"id"},
				TargetType: "Profile",
			},
			expected: `profile: Profile @relation(type: ONE_TO_ONE, fields: ["userId"], references: ["id"])`,
		},
		{
			name: "one_to_many",
			relation: &Relation{
				FieldName:  "posts",
				Type:       schemapkg.OneToMany,
				Fields:     []string{"id"},
				References: []string{"userId"},
				TargetType: "Post",
			},
			expected: `posts: [Post] @relation(type: ONE_TO_MANY, fields: ["id"], references: ["userId"])`,
		},
		{
			name: "many_to_many",
			relation: &Relation{
				FieldName:            "categories",
				Type:                 schemapkg.ManyToMany,
				Fields:               []string{"id"},
				References:           []string{"id"},
				TargetType:           "Category",
				ManyToManyTable:      "posts_to_categories",
				ManyToManyFields:     []string{"post_id"},
				ManyToManyReferences: []string{"category_id"},
			},
			expected: `categories: [Category] @relation(type: MANY_TO_MANY, fields: ["id"], references: ["id"], manyToManyTable: "posts_to_categories", manyToManyFields: ["post_id"], manyToManyReferences: ["category_id"])`,
		},
		{
			name: "many_to_many_without_optional_fields",
			relation: &Relation{
				FieldName:  "tags",
				Type:       schemapkg.ManyToMany,
				Fields:     []string{"id"},
				References: []string{"id"},
				TargetType: "Tag",
			},
			expected: `tags: [Tag] @relation(type: MANY_TO_MANY, fields: ["id"], references: ["id"])`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := relationToGraphQL(tt.relation)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateSchema_QueryFields(t *testing.T) {
	tests := []struct {
		name       string
		queryField *QueryField
		expected   string
	}{
		{
			name:       "list_with_generate",
			queryField: &QueryField{Name: "users", Type: "User", IsList: true, Generate: true},
			expected:   "users: [User] @generate",
		},
		{
			name:       "single_without_generate",
			queryField: &QueryField{Name: "user", Type: "User", IsList: false, Generate: false},
			expected:   "user: User",
		},
		{
			name:       "list_without_generate",
			queryField: &QueryField{Name: "users", Type: "User", IsList: true, Generate: false},
			expected:   "users: [User]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := qfToGraphQL(tt.queryField)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// normalizeWhitespace removes extra whitespace and normalizes line endings
func normalizeWhitespace(s string) string {
	// Replace all sequences of whitespace with a single space
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")

	// Split by lines and trim each line
	lines := strings.Split(s, "\n")
	var normalized []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}
	return strings.Join(normalized, "\n")
}
