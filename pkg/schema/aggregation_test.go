package schema

import (
	"fmt"
	"testing"

	"github.com/iancoleman/strcase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
)

func Test_AugmentAggregation(t *testing.T) {
	var testCases = []generateTestCase{
		{
			name:                      "aggregation_base",
			baseSchemaFile:            "testdata/base.graphql",
			expectedSchemaFile:        "testdata/aggregation_expected.graphql",
			expectedFastgqlSchemaFile: "testdata/aggregation_fastgql_expected.graphql",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			generateTestRunner(t, &tc, AggregationAugmenter)
		})
	}
}

// Test_AggregationAugmenter_Unit tests the AggregationAugmenter function in isolation
func Test_AggregationAugmenter_Unit(t *testing.T) {
	tests := []struct {
		name                 string
		schemaDefinition     string
		queryField           string
		expectAggregateField bool
		aggregateFieldName   string
	}{
		{
			name: "adds_aggregate_field_when_directive_present",
			schemaDefinition: `
				type User {
					id: ID!
					name: String!
				}
				type Query {
					users: [User] @generate(aggregate: true)
				}
			`,
			queryField:           "users",
			expectAggregateField: true,
			aggregateFieldName:   "_usersAggregate",
		},
		{
			name: "skips_aggregate_when_directive_false",
			schemaDefinition: `
				type User {
					id: ID!
					name: String!
				}
				type Query {
					users: [User] @generate(aggregate: false)
				}
			`,
			queryField:           "users",
			expectAggregateField: false,
		},
		{
			name: "skips_non_list_fields",
			schemaDefinition: `
				type User {
					id: ID!
				}
				type Query {
					user: User @generate(aggregate: true)
				}
			`,
			queryField:           "user",
			expectAggregateField: false,
		},
		{
			name: "skips_fields_without_directive",
			schemaDefinition: `
				type User {
					id: ID!
				}
				type Query {
					users: [User]
				}
			`,
			queryField:           "users",
			expectAggregateField: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := buildTestSchema(t, tt.schemaDefinition)

			err := AggregationAugmenter(schema)
			require.NoError(t, err)

			field := schema.Query.Fields.ForName(tt.queryField)
			require.NotNil(t, field, "Query field should exist")

			if tt.expectAggregateField {
				aggField := schema.Query.Fields.ForName(tt.aggregateFieldName)
				assert.NotNil(t, aggField, "Aggregate field should be created")
				if aggField != nil {
					// Check that it has the generate directive for filter
					assert.NotNil(t, aggField.Directives.ForName(generateDirectiveName))
					// Check groupBy argument exists
					groupByArg := aggField.Arguments.ForName("groupBy")
				assert.NotNil(t, groupByArg, "groupBy argument should exist")
			}
		} else if tt.aggregateFieldName != "" {
			aggField := schema.Query.Fields.ForName(tt.aggregateFieldName)
			assert.Nil(t, aggField, "Aggregate field should not be created")
		}
		})
	}
}

// Test_addAggregateObject tests the aggregate object creation
func Test_addAggregateObject(t *testing.T) {
	tests := []struct {
		name              string
		schemaDefinition  string
		objectType        string
		expectedAggregate string
		expectCount       bool
		expectGroup       bool
		expectSum         bool
		expectAvg         bool
		expectMin         bool
		expectMax         bool
	}{
		{
			name: "creates_aggregate_with_all_fields",
			schemaDefinition: `
				type User {
					id: ID!
					name: String!
					age: Int!
					score: Float!
				}
			`,
			objectType:        "User",
			expectedAggregate: "UsersAggregate",
			expectCount:       true,
			expectGroup:       true,
			expectSum:         true,
			expectAvg:         true,
			expectMin:         true,
			expectMax:         true,
		},
		{
			name: "returns_existing_aggregate_if_already_created",
			schemaDefinition: `
				type User {
					id: ID!
					name: String!
				}
				type UsersAggregate {
					count: Int!
				}
			`,
			objectType:        "User",
			expectedAggregate: "UsersAggregate",
			expectCount:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := buildTestSchema(t, tt.schemaDefinition)
			objDef := schema.Types[tt.objectType]
			require.NotNil(t, objDef)

			result := addAggregateObject(schema, objDef)
			require.NotNil(t, result, "Aggregate object should be created")
			assert.Equal(t, tt.expectedAggregate, result.Name)

			// Check aggregate type exists in schema
			aggType, exists := schema.Types[tt.expectedAggregate]
			require.True(t, exists, "Aggregate type should be in schema")

			// Check expected fields
			if tt.expectCount {
				assert.NotNil(t, aggType.Fields.ForName("count"), "count field should exist")
			}
			if tt.expectGroup {
				assert.NotNil(t, aggType.Fields.ForName("group"), "group field should exist")
			}
			if tt.expectSum {
				assert.NotNil(t, aggType.Fields.ForName("sum"), "sum field should exist")
			}
			if tt.expectAvg {
				assert.NotNil(t, aggType.Fields.ForName("avg"), "avg field should exist")
			}
			if tt.expectMin {
				assert.NotNil(t, aggType.Fields.ForName("min"), "min field should exist")
			}
			if tt.expectMax {
				assert.NotNil(t, aggType.Fields.ForName("max"), "max field should exist")
			}
		})
	}
}

// Test_addAggregationFieldToSchema tests aggregate field type creation
func Test_addAggregationFieldToSchema(t *testing.T) {
	tests := []struct {
		name           string
		aggregateType  aggregate
		objectFields   []fieldInfo
		expectFields   []string
		unexpectedFields []string
	}{
		{
			name: "sum_includes_only_numeric_types",
			aggregateType: aggregate{
				name:               "sum",
				allowedScalarTypes: []string{"Int", "Float"},
				kind:               "Float",
			},
			objectFields: []fieldInfo{
				{name: "id", typeName: "ID", isList: false},
				{name: "age", typeName: "Int", isList: false},
				{name: "score", typeName: "Float", isList: false},
				{name: "name", typeName: "String", isList: false},
			},
			expectFields:     []string{"age", "score"},
			unexpectedFields: []string{"id", "name"},
		},
		{
			name: "avg_includes_only_numeric_types",
			aggregateType: aggregate{
				name:               "avg",
				allowedScalarTypes: []string{"Int", "Float"},
				kind:               "Float",
			},
			objectFields: []fieldInfo{
				{name: "age", typeName: "Int", isList: false},
				{name: "name", typeName: "String", isList: false},
			},
			expectFields:     []string{"age"},
			unexpectedFields: []string{"name"},
		},
		{
			name: "min_includes_multiple_types",
			aggregateType: aggregate{
				name:               "min",
				allowedScalarTypes: []string{"Int", "Float", "String", "DateTime", "ID"},
			},
			objectFields: []fieldInfo{
				{name: "id", typeName: "ID", isList: false},
				{name: "age", typeName: "Int", isList: false},
				{name: "name", typeName: "String", isList: false},
				{name: "active", typeName: "Boolean", isList: false},
			},
			expectFields:     []string{"id", "age", "name"},
			unexpectedFields: []string{"active"},
		},
		{
			name: "skips_list_fields",
			aggregateType: aggregate{
				name:               "sum",
				allowedScalarTypes: []string{"Int", "Float"},
				kind:               "Float",
			},
			objectFields: []fieldInfo{
				{name: "age", typeName: "Int", isList: false},
				{name: "scores", typeName: "Int", isList: true},
			},
			expectFields:     []string{"age"},
			unexpectedFields: []string{"scores"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build schema with object type
			schema := buildSchemaWithFields(t, "TestObject", tt.objectFields)
			objDef := schema.Types["TestObject"]

			result := addAggregationFieldToSchema(schema, objDef, tt.aggregateType)

			if len(tt.expectFields) == 0 {
				assert.Nil(t, result, "Should return nil when no valid fields")
				return
			}

			require.NotNil(t, result, "Should return aggregate field definition")

			// Check the aggregate type was created in schema
			aggTypeName := fmt.Sprintf("_TestObject%s", strcase.ToCamel(tt.aggregateType.name))
			aggType, exists := schema.Types[aggTypeName]
			require.True(t, exists, "Aggregate type should be created in schema")

			// Check expected fields exist
			for _, fieldName := range tt.expectFields {
				field := aggType.Fields.ForName(fieldName)
				assert.NotNil(t, field, "Expected field %s to exist", fieldName)
			}

			// Check unexpected fields don't exist
			for _, fieldName := range tt.unexpectedFields {
				field := aggType.Fields.ForName(fieldName)
				assert.Nil(t, field, "Did not expect field %s", fieldName)
			}
		})
	}
}

// Test_scalarAllowed tests the scalar type checking
func Test_scalarAllowed(t *testing.T) {
	tests := []struct {
		scalar  string
		allowed []string
		want    bool
	}{
		{scalar: "Int", allowed: []string{"Int", "Float"}, want: true},
		{scalar: "Float", allowed: []string{"Int", "Float"}, want: true},
		{scalar: "String", allowed: []string{"Int", "Float"}, want: false},
		{scalar: "ID", allowed: []string{"Int", "Float", "String", "DateTime", "ID"}, want: true},
		{scalar: "DateTime", allowed: []string{"Int", "Float", "String", "DateTime", "ID"}, want: true},
		{scalar: "Boolean", allowed: []string{"Int", "Float"}, want: false},
		{scalar: "CustomScalar", allowed: []string{"Int", "Float"}, want: false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_in_%v", tt.scalar, tt.allowed), func(t *testing.T) {
			got := scalarAllowed(tt.scalar, tt.allowed)
			assert.Equal(t, tt.want, got)
		})
	}
}

// Test_addAggregateGroupByObject tests groupBy enum creation
func Test_addAggregateGroupByObject(t *testing.T) {
	tests := []struct {
		name           string
		schemaDefinition string
		objectType     string
		expectedEnum   string
		expectedValues []string
		skipValues     []string
	}{
		{
			name: "creates_groupby_enum_for_scalar_fields",
			schemaDefinition: `
				type User {
					id: ID!
					name: String!
					age: Int!
				}
			`,
			objectType:     "User",
			expectedEnum:   "UserGroupBy",
			expectedValues: []string{"ID", "NAME", "AGE"},
		},
		{
			name: "skips_list_fields",
			schemaDefinition: `
				type User {
					id: ID!
					tags: [String]
					posts: [Post]
				}
				type Post {
					id: ID!
				}
			`,
			objectType:     "User",
			expectedEnum:   "UserGroupBy",
			expectedValues: []string{"ID"},
			skipValues:     []string{"TAGS", "POSTS"},
		},
		{
			name: "skips_composite_fields",
			schemaDefinition: `
				type User {
					id: ID!
					profile: Profile
				}
				type Profile {
					bio: String
				}
			`,
			objectType:     "User",
			expectedEnum:   "UserGroupBy",
			expectedValues: []string{"ID"},
			skipValues:     []string{"PROFILE"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := buildTestSchema(t, tt.schemaDefinition)
			objDef := schema.Types[tt.objectType]
			require.NotNil(t, objDef)

			// Call the function
			addAggregateGroupByObject(schema, objDef)

			// Check enum was created
			enumDef, exists := schema.Types[tt.expectedEnum]
			require.True(t, exists, "GroupBy enum should be created")
			assert.Equal(t, ast.Enum, enumDef.Kind)

			// Check expected values
			for _, value := range tt.expectedValues {
				found := false
				for _, enumVal := range enumDef.EnumValues {
					if enumVal.Name == value {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected enum value %s", value)
			}

			// Check values that should be skipped
			for _, value := range tt.skipValues {
				found := false
				for _, enumVal := range enumDef.EnumValues {
					if enumVal.Name == value {
						found = true
						break
					}
				}
				assert.False(t, found, "Did not expect enum value %s", value)
			}
		})
	}
}

// Test_addRecursiveAggregation tests recursive aggregation on relations
func Test_addRecursiveAggregation(t *testing.T) {
	tests := []struct {
		name              string
		schemaDefinition  string
		objectType        string
		expectAggFields   []string
	}{
		{
			name: "adds_aggregate_for_relation_fields",
			schemaDefinition: `
				type User {
					id: ID!
					posts: [Post] @relation(type: ONE_TO_MANY, fields: ["id"], references: ["user_id"])
				}
				type Post {
					id: ID!
					title: String!
				}
			`,
			objectType:      "User",
			expectAggFields: []string{"_postsAggregate"},
		},
		{
			name: "skips_non_relation_fields",
			schemaDefinition: `
				type User {
					id: ID!
					posts: [Post]
				}
				type Post {
					id: ID!
				}
			`,
			objectType:      "User",
			expectAggFields: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := buildTestSchema(t, tt.schemaDefinition)
			objDef := schema.Types[tt.objectType]
			require.NotNil(t, objDef)

			// First create the groupBy (required by addRecursiveAggregation)
			addAggregateGroupByObject(schema, objDef)

			// Call the function
			addRecursiveAggregation(schema, objDef)

			// Check expected aggregate fields
			for _, fieldName := range tt.expectAggFields {
				field := objDef.Fields.ForName(fieldName)
				assert.NotNil(t, field, "Expected aggregate field %s", fieldName)
			}

			// If no fields expected, verify none were added
			if len(tt.expectAggFields) == 0 {
				for _, field := range objDef.Fields {
					assert.False(t, field.Name == "_postsAggregate", "Should not add aggregate field")
				}
			}
		})
	}
}

// Helper types and functions for testing

type fieldInfo struct {
	name     string
	typeName string
	isList   bool
}

// buildSchemaWithFields creates a test schema with specified fields
func buildSchemaWithFields(t *testing.T, typeName string, fields []fieldInfo) *ast.Schema {
	t.Helper()

	sdl := fmt.Sprintf("type %s {\n", typeName)
	for _, f := range fields {
		if f.isList {
			sdl += fmt.Sprintf("  %s: [%s]\n", f.name, f.typeName)
		} else {
			sdl += fmt.Sprintf("  %s: %s\n", f.name, f.typeName)
		}
	}
	sdl += "}\n"

	return buildTestSchema(t, sdl)
}

