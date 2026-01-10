package postgres

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/roneli/fastgql/pkg/execution/testhelpers"
	"github.com/roneli/fastgql/pkg/importer"
	schemapkg "github.com/roneli/fastgql/pkg/schema"
)

type testCase struct {
	name     string
	sqlFile  string
	options  importer.IntrospectOptions
	validate func(t *testing.T, schema *importer.Schema)
}

func TestIntrospect(t *testing.T) {
	tests := []testCase{
		{
			name:    "basic_types",
			sqlFile: "basic_types.sql",
			options: importer.IntrospectOptions{
				SchemaName:      "public",
				Tables:          []string{"users"},
				GenerateQueries: true,
				GenerateFilters: true,
			},
			validate: func(t *testing.T, schema *importer.Schema) {
				// Verify users table
				usersType := findObjectType(schema.ObjectTypes, "Users")
				require.NotNil(t, usersType, "should have Users type")
				assert.Equal(t, "users", usersType.TableName)
				assert.True(t, usersType.GenerateFilterInput)

				// Verify fields
				assertField(t, usersType.Fields, "id", schemapkg.GraphQLTypeInt, true, false)
				assertField(t, usersType.Fields, "name", schemapkg.GraphQLTypeString, true, false)
				assertField(t, usersType.Fields, "email", schemapkg.GraphQLTypeString, false, false)
				assertField(t, usersType.Fields, "age", schemapkg.GraphQLTypeInt, false, false)
				assertField(t, usersType.Fields, "active", schemapkg.GraphQLTypeBoolean, false, false)
				assertField(t, usersType.Fields, "balance", schemapkg.GraphQLTypeFloat, false, false)

				// Verify query fields
				require.Len(t, schema.QueryFields, 1)
				assert.Equal(t, "users", schema.QueryFields[0].Name)
				assert.Equal(t, "Users", schema.QueryFields[0].Type)
				assert.True(t, schema.QueryFields[0].IsList)
				assert.True(t, schema.QueryFields[0].Generate)
			},
		},
		{
			name:    "json_types",
			sqlFile: "json_types.sql",
			options: importer.IntrospectOptions{
				SchemaName: "public",
				Tables:     []string{"products"},
			},
			validate: func(t *testing.T, schema *importer.Schema) {
				productsType := findObjectType(schema.ObjectTypes, "Products")
				require.NotNil(t, productsType)

				// Verify JSON fields
				attributesField := findField(productsType.Fields, "attributes")
				require.NotNil(t, attributesField)
				assert.True(t, attributesField.IsJSON)
				assert.Equal(t, schemapkg.GraphQLTypeMap, attributesField.Type)

				metadataField := findField(productsType.Fields, "metadata")
				require.NotNil(t, metadataField)
				assert.True(t, metadataField.IsJSON)
				assert.Equal(t, schemapkg.GraphQLTypeMap, metadataField.Type)

				// Verify array field
				tagsField := findField(productsType.Fields, "tags")
				require.NotNil(t, tagsField)
				assert.True(t, tagsField.IsList)
			},
		},
		{
			name:    "one_to_one",
			sqlFile: "one_to_one.sql",
			options: importer.IntrospectOptions{
				SchemaName: "public",
				Tables:     []string{"users", "profiles"},
			},
			validate: func(t *testing.T, schema *importer.Schema) {
				profilesType := findObjectType(schema.ObjectTypes, "Profiles")
				require.NotNil(t, profilesType)

				// Verify ONE_TO_ONE relation
				require.Len(t, profilesType.Relations, 1)
				relation := profilesType.Relations[0]
				assert.Equal(t, "user", relation.FieldName)
				assert.Equal(t, schemapkg.OneToOne, relation.Type)
				assert.Equal(t, "Users", relation.TargetType)
				assert.Equal(t, []string{"userId"}, relation.Fields)
				assert.Equal(t, []string{"id"}, relation.References)
			},
		},
		{
			name:    "one_to_many",
			sqlFile: "one_to_many.sql",
			options: importer.IntrospectOptions{
				SchemaName: "public",
				Tables:     []string{"users", "posts"},
			},
			validate: func(t *testing.T, schema *importer.Schema) {
				usersType := findObjectType(schema.ObjectTypes, "Users")
				require.NotNil(t, usersType)

				// Verify ONE_TO_MANY relation
				require.Len(t, usersType.Relations, 1)
				relation := usersType.Relations[0]
				assert.Equal(t, "posts", relation.FieldName)
				assert.Equal(t, schemapkg.OneToMany, relation.Type)
				assert.Equal(t, "Posts", relation.TargetType)
				assert.Equal(t, []string{"id"}, relation.Fields)
				assert.Equal(t, []string{"userId"}, relation.References)
			},
		},
		{
			name:    "many_to_many",
			sqlFile: "many_to_many.sql",
			options: importer.IntrospectOptions{
				SchemaName: "public",
				Tables:     []string{"posts", "categories", "posts_categories"},
			},
			validate: func(t *testing.T, schema *importer.Schema) {
				postsType := findObjectType(schema.ObjectTypes, "Posts")
				require.NotNil(t, postsType)

				// Verify MANY_TO_MANY relation
				require.Len(t, postsType.Relations, 1)
				relation := postsType.Relations[0]
				assert.Equal(t, "categories", relation.FieldName)
				assert.Equal(t, schemapkg.ManyToMany, relation.Type)
				assert.Equal(t, "Categories", relation.TargetType)
				assert.Equal(t, "posts_categories", relation.ManyToManyTable)
			},
		},
		{
			name:    "mixed",
			sqlFile: "mixed.sql",
			options: importer.IntrospectOptions{
				SchemaName:      "public",
				Tables:          []string{"users", "posts", "categories", "posts_categories", "profiles"},
				GenerateQueries: true,
				GenerateFilters: true,
			},
			validate: func(t *testing.T, schema *importer.Schema) {
				// Verify all tables are present
				assert.Len(t, schema.ObjectTypes, 5) // users, posts, categories, posts_categories, profiles

				usersType := findObjectType(schema.ObjectTypes, "Users")
				require.NotNil(t, usersType)
				assert.True(t, usersType.GenerateFilterInput)
				assertField(t, usersType.Fields, "metadata", schemapkg.GraphQLTypeMap, false, false) // JSONB is not an array
				assertField(t, usersType.Fields, "tags", schemapkg.GraphQLTypeString, false, true)   // array

				// Verify relations
				postsType := findObjectType(schema.ObjectTypes, "Posts")
				require.NotNil(t, postsType)
				require.Len(t, postsType.Relations, 2) // user (ONE_TO_MANY) and categories (MANY_TO_MANY)

				profilesType := findObjectType(schema.ObjectTypes, "Profiles")
				require.NotNil(t, profilesType)
				require.Len(t, profilesType.Relations, 1) // user (ONE_TO_ONE)

				// Verify query fields
				assert.Len(t, schema.QueryFields, 5)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			pool, cleanup, err := setupTestDB(ctx, tt.sqlFile)
			require.NoError(t, err)
			defer cleanup()

			source := NewSource(pool)
			schema, err := source.Introspect(ctx, tt.options)
			require.NoError(t, err)
			require.NotNil(t, schema)

			tt.validate(t, schema)
		})
	}
}

// setupTestDB creates a test database with the given init SQL file
func setupTestDB(ctx context.Context, initSQLFile string) (*pgxpool.Pool, func(), error) {
	// Use testhelpers to get a pool
	pool, cleanup, err := testhelpers.GetTestPostgresPool(ctx)
	if err != nil {
		return nil, nil, err
	}

	// Read and execute init SQL
	_, b, _, _ := runtime.Caller(0)
	basepath := filepath.Dir(b)
	testdataPath := filepath.Join(basepath, "testdata", initSQLFile)

	sqlBytes, err := os.ReadFile(testdataPath)
	if err != nil {
		cleanup()
		return nil, nil, err
	}

	sql := string(sqlBytes)

	// Remove comment lines and split into statements
	lines := strings.Split(sql, "\n")
	cleanedLines := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Skip empty lines and comment-only lines
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}
		cleanedLines = append(cleanedLines, line)
	}
	cleanedSQL := strings.Join(cleanedLines, "\n")

	// Split by semicolon and execute each statement
	statements := strings.Split(cleanedSQL, ";")
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		// Execute the statement
		if _, err := pool.Exec(ctx, stmt); err != nil {
			cleanup()
			return nil, nil, fmt.Errorf("failed to execute SQL statement: %w\nStatement: %s", err, stmt)
		}
	}

	return pool, cleanup, nil
}

// Helper functions
func findObjectType(types []*importer.ObjectType, name string) *importer.ObjectType {
	for _, t := range types {
		if t.Name == name {
			return t
		}
	}
	return nil
}

func findField(fields []*importer.Field, name string) *importer.Field {
	for _, f := range fields {
		if f.Name == name {
			return f
		}
	}
	return nil
}

func assertField(t *testing.T, fields []*importer.Field, name, expectedType string, expectedNonNull, expectedList bool) {
	field := findField(fields, name)
	require.NotNil(t, field, "field %s should exist", name)
	assert.Equal(t, expectedType, field.Type, "field %s should have type %s", name, expectedType)
	assert.Equal(t, expectedNonNull, field.IsNonNull, "field %s IsNonNull should be %v", name, expectedNonNull)
	assert.Equal(t, expectedList, field.IsList, "field %s IsList should be %v", name, expectedList)
}
