package sql

import (
	"testing"

	"github.com/doug-martin/goqu/v9"
	"github.com/stretchr/testify/assert"
	"github.com/vektah/gqlparser/v2/ast"
)

func TestTableDefinition_TableExpression(t *testing.T) {
	tests := []struct {
		name        string
		tableDef    tableDefinition
		expectedSQL string
	}{
		{
			name:        "table_without_schema",
			tableDef:    tableDefinition{name: "users", schema: ""},
			expectedSQL: `SELECT * FROM "users"`,
		},
		{
			name:        "table_with_schema",
			tableDef:    tableDefinition{name: "users", schema: "public"},
			expectedSQL: `SELECT * FROM "public"."users"`,
		},
		{
			name:        "table_with_app_schema",
			tableDef:    tableDefinition{name: "posts", schema: "app"},
			expectedSQL: `SELECT * FROM "app"."posts"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr := tt.tableDef.TableExpression()
			sql, _, err := goqu.Dialect("postgres").From(expr).ToSQL()
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedSQL, sql)
		})
	}
}

func TestTableDefinition_String(t *testing.T) {
	tests := []struct {
		name     string
		tableDef tableDefinition
		expected string
	}{
		{
			name:     "without_schema",
			tableDef: tableDefinition{name: "users", schema: ""},
			expected: `"users"`,
		},
		{
			name:     "with_schema",
			tableDef: tableDefinition{name: "users", schema: "public"},
			expected: `"public"."users"`,
		},
		{
			name:     "with_app_schema",
			tableDef: tableDefinition{name: "posts", schema: "app"},
			expected: `"app"."posts"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.tableDef.String()
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestGetTableName(t *testing.T) {
	tests := []struct {
		name           string
		schema         *ast.Schema
		typeName       string
		fieldName      string
		expectedName   string
		expectedSchema string
		expectObjType  bool
	}{
		{
			name: "type_with_table_directive_name_only",
			schema: &ast.Schema{
				Types: map[string]*ast.Definition{
					"User": {
						Name: "User",
						Directives: ast.DirectiveList{
							{
								Name: "table",
								Arguments: ast.ArgumentList{
									{Name: "name", Value: &ast.Value{Raw: "users"}},
								},
							},
						},
					},
				},
			},
			typeName:       "User",
			fieldName:      "users",
			expectedName:   "users",
			expectedSchema: "",
			expectObjType:  true,
		},
		{
			name: "type_with_table_directive_name_and_schema",
			schema: &ast.Schema{
				Types: map[string]*ast.Definition{
					"Post": {
						Name: "Post",
						Directives: ast.DirectiveList{
							{
								Name: "table",
								Arguments: ast.ArgumentList{
									{Name: "name", Value: &ast.Value{Raw: "posts"}},
									{Name: "schema", Value: &ast.Value{Raw: "blog"}},
								},
							},
						},
					},
				},
			},
			typeName:       "Post",
			fieldName:      "posts",
			expectedName:   "posts",
			expectedSchema: "blog",
			expectObjType:  true,
		},
		{
			name: "type_without_table_directive",
			schema: &ast.Schema{
				Types: map[string]*ast.Definition{
					"Category": {
						Name:       "Category",
						Directives: ast.DirectiveList{},
					},
				},
			},
			typeName:       "Category",
			fieldName:      "categories",
			expectedName:   "category", // lowercase of type name
			expectedSchema: "",
			expectObjType:  true,
		},
		{
			name:           "type_not_in_schema",
			schema:         &ast.Schema{Types: map[string]*ast.Definition{}},
			typeName:       "Unknown",
			fieldName:      "unknowns",
			expectedName:   "unknowns", // falls back to field name
			expectedSchema: "",
			expectObjType:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getTableName(tt.schema, tt.typeName, tt.fieldName)
			assert.Equal(t, tt.expectedName, got.name)
			assert.Equal(t, tt.expectedSchema, got.schema)
			if tt.expectObjType {
				assert.NotNil(t, got.objType)
			} else {
				assert.Nil(t, got.objType)
			}
		})
	}
}

func TestGetTableNameFromField(t *testing.T) {
	tests := []struct {
		name           string
		schema         *ast.Schema
		fieldDef       *ast.FieldDefinition
		expectedName   string
		expectedSchema string
	}{
		{
			name: "field_with_table_directive",
			schema: &ast.Schema{
				Types: map[string]*ast.Definition{
					"User": {
						Name: "User",
						Directives: ast.DirectiveList{
							{
								Name: "table",
								Arguments: ast.ArgumentList{
									{Name: "name", Value: &ast.Value{Raw: "app_users"}},
									{Name: "schema", Value: &ast.Value{Raw: "public"}},
								},
							},
						},
					},
				},
			},
			fieldDef: &ast.FieldDefinition{
				Name: "user",
				Type: &ast.Type{NamedType: "User"},
			},
			expectedName:   "app_users",
			expectedSchema: "public",
		},
		{
			name: "field_without_table_directive",
			schema: &ast.Schema{
				Types: map[string]*ast.Definition{
					"Comment": {
						Name:       "Comment",
						Directives: ast.DirectiveList{},
					},
				},
			},
			fieldDef: &ast.FieldDefinition{
				Name: "comments",
				Type: &ast.Type{NamedType: "Comment"},
			},
			expectedName:   "comment",
			expectedSchema: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getTableNameFromField(tt.schema, tt.fieldDef)
			assert.Equal(t, tt.expectedName, got.name)
			assert.Equal(t, tt.expectedSchema, got.schema)
		})
	}
}

func TestGetAggregateTableName(t *testing.T) {
	postsDef := &ast.Definition{
		Name: "Post",
		Directives: ast.DirectiveList{
			{
				Name: "table",
				Arguments: ast.ArgumentList{
					{Name: "name", Value: &ast.Value{Raw: "posts"}},
					{Name: "schema", Value: &ast.Value{Raw: "blog"}},
				},
			},
		},
	}

	queryDef := &ast.Definition{
		Name: "Query",
		Fields: ast.FieldList{
			{
				Name: "posts",
				Type: &ast.Type{NamedType: "Post"},
			},
		},
	}

	schema := &ast.Schema{
		Types: map[string]*ast.Definition{
			"Post":  postsDef,
			"Query": queryDef,
		},
	}

	tests := []struct {
		name           string
		field          *ast.Field
		expectedName   string
		expectedSchema string
	}{
		{
			name: "posts_aggregate",
			field: &ast.Field{
				Name:             "_postsAggregate",
				ObjectDefinition: queryDef,
			},
			expectedName:   "posts",
			expectedSchema: "blog",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getAggregateTableName(schema, tt.field)
			assert.Equal(t, tt.expectedName, got.name)
			assert.Equal(t, tt.expectedSchema, got.schema)
		})
	}
}

func TestGetTableNamePrefix(t *testing.T) {
	userDef := &ast.Definition{
		Name: "User",
		Directives: ast.DirectiveList{
			{
				Name: "table",
				Arguments: ast.ArgumentList{
					{Name: "name", Value: &ast.Value{Raw: "users"}},
					{Name: "schema", Value: &ast.Value{Raw: "app"}},
				},
			},
		},
	}

	schema := &ast.Schema{
		Types: map[string]*ast.Definition{
			"User": userDef,
		},
	}

	tests := []struct {
		name           string
		prefix         string
		field          *ast.Field
		expectedName   string
		expectedSchema string
	}{
		{
			name:           "create_users",
			prefix:         "create",
			field:          &ast.Field{Name: "createUsers"},
			expectedName:   "users",
			expectedSchema: "app",
		},
		{
			name:           "delete_users",
			prefix:         "delete",
			field:          &ast.Field{Name: "deleteUsers"},
			expectedName:   "users",
			expectedSchema: "app",
		},
		{
			name:           "update_users",
			prefix:         "update",
			field:          &ast.Field{Name: "updateUsers"},
			expectedName:   "users",
			expectedSchema: "app",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getTableNamePrefix(schema, tt.prefix, tt.field)
			assert.Equal(t, tt.expectedName, got.name)
			assert.Equal(t, tt.expectedSchema, got.schema)
		})
	}
}

