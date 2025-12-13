package importer_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/parser"

	"github.com/roneli/fastgql/pkg/execution/testhelpers"
	"github.com/roneli/fastgql/pkg/importer"
	"github.com/roneli/fastgql/pkg/importer/postgres"
	schemapkg "github.com/roneli/fastgql/pkg/schema"
)

func TestGeneratorE2E(t *testing.T) {
	ctx := context.Background()

	// Setup test database
	pool, cleanup, err := setupTestDB(ctx, "mixed.sql")
	require.NoError(t, err)
	defer cleanup()

	// Create PostgreSQL source
	source := postgres.NewSource(pool)

	// Test with all options enabled
	options := importer.IntrospectOptions{
		SchemaName:      "public",
		Tables:          []string{"users", "posts", "categories", "posts_categories", "profiles"},
		GenerateQueries: true,
		GenerateFilters: true,
	}

	// Introspect database
	schema, err := source.Introspect(ctx, options)
	require.NoError(t, err)
	require.NotNil(t, schema)

	// Validate introspected schema structure
	require.Len(t, schema.ObjectTypes, 5, "should have 5 object types")
	require.Len(t, schema.QueryFields, 5, "should have 5 query fields")

	// Find specific types
	usersType := findObjectType(schema.ObjectTypes, "Users")
	require.NotNil(t, usersType, "should have Users type")
	assert.Equal(t, "users", usersType.TableName)
	assert.True(t, usersType.GenerateFilterInput)

	postsType := findObjectType(schema.ObjectTypes, "Posts")
	require.NotNil(t, postsType, "should have Posts type")
	assert.Equal(t, "posts", postsType.TableName)

	// Validate fields
	assertField(t, usersType.Fields, "id", schemapkg.GraphQLTypeInt, true, false)
	assertField(t, usersType.Fields, "name", schemapkg.GraphQLTypeString, true, false)
	assertField(t, usersType.Fields, "email", schemapkg.GraphQLTypeString, true, false)
	assertField(t, usersType.Fields, "metadata", schemapkg.GraphQLTypeMap, false, false)
	assertField(t, usersType.Fields, "tags", schemapkg.GraphQLTypeString, false, true) // array

	// Validate relations
	require.Len(t, postsType.Relations, 2, "Posts should have 2 relations")

	// Check for user relation (ONE_TO_ONE from posts to users)
	userRelation := findRelation(postsType.Relations, "user")
	require.NotNil(t, userRelation, "Posts should have user relation")
	assert.Equal(t, schemapkg.OneToOne, userRelation.Type)
	assert.Equal(t, "Users", userRelation.TargetType)

	// Check for categories relation (MANY_TO_MANY)
	categoriesRelation := findRelation(postsType.Relations, "categories")
	require.NotNil(t, categoriesRelation, "Posts should have categories relation")
	assert.Equal(t, schemapkg.ManyToMany, categoriesRelation.Type)
	assert.Equal(t, "Categories", categoriesRelation.TargetType)
	assert.Equal(t, "posts_categories", categoriesRelation.ManyToManyTable)

	// Check users has posts relation (ONE_TO_MANY)
	require.Len(t, usersType.Relations, 1, "Users should have 1 relation")
	postsRelation := findRelation(usersType.Relations, "posts")
	require.NotNil(t, postsRelation, "Users should have posts relation")
	assert.Equal(t, schemapkg.OneToMany, postsRelation.Type)
	assert.Equal(t, "Posts", postsRelation.TargetType)

	// Check profiles has user relation (ONE_TO_ONE)
	profilesType := findObjectType(schema.ObjectTypes, "Profiles")
	require.NotNil(t, profilesType, "should have Profiles type")
	require.Len(t, profilesType.Relations, 1, "Profiles should have 1 relation")
	profileUserRelation := findRelation(profilesType.Relations, "user")
	require.NotNil(t, profileUserRelation, "Profiles should have user relation")
	assert.Equal(t, schemapkg.OneToOne, profileUserRelation.Type)

	// Generate GraphQL schema
	astSource, err := importer.GenerateSchema(schema)
	require.NoError(t, err)
	require.NotNil(t, astSource)

	// Validate generated schema is valid GraphQL
	_, err = parser.ParseSchema(astSource)
	require.NoError(t, err, "generated schema should be valid GraphQL")

	// Read expected schema file
	_, b, _, _ := runtime.Caller(0)
	basepath := filepath.Dir(b)
	expectedSchemaPath := filepath.Join(basepath, "testdata", "generator_e2e_expected.graphql")
	expectedSchema, err := os.ReadFile(expectedSchemaPath)
	require.NoError(t, err, "failed to read expected schema file")

	// Compare generated schema with expected (normalize whitespace)
	generatedSchema := astSource.Input
	expectedSchemaStr := string(expectedSchema)

	// Normalize whitespace for comparison
	normalizedGenerated := removeWhitespaceWithRegex(generatedSchema)
	normalizedExpected := removeWhitespaceWithRegex(expectedSchemaStr)

	if !assert.Equal(t, normalizedExpected, normalizedGenerated, "generated schema should match expected schema") {
		// Print actual output for debugging
		fmt.Println("=== Generated schema ===")
		fmt.Println(generatedSchema)
		fmt.Println("\n=== Expected schema ===")
		fmt.Println(expectedSchemaStr)
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
	testdataPath := filepath.Join(basepath, "postgres", "testdata", initSQLFile)

	sqlBytes, err := os.ReadFile(testdataPath)
	if err != nil {
		cleanup()
		return nil, nil, err
	}

	sql := string(sqlBytes)

	// Remove comment lines and split into statements
	lines := strings.Split(sql, "\n")
	var cleanedLines []string
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

func findRelation(relations []*importer.Relation, name string) *importer.Relation {
	for _, r := range relations {
		if r.FieldName == name {
			return r
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

// removeWhitespaceWithRegex normalizes whitespace for comparison (same as schema package tests)
func removeWhitespaceWithRegex(s string) string {
	reg := regexp.MustCompile(`[\s]+`) // Match any whitespace character
	return reg.ReplaceAllString(s, "")
}
