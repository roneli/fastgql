package schema

import (
	"testing"

	"github.com/iancoleman/strcase"
	"github.com/jinzhu/inflection"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
)

func Test_AugmentMutations(t *testing.T) {
	var testCases = []generateTestCase{
		{
			name:                      "mutations_base",
			baseSchemaFile:            "testdata/mutations.graphql",
			expectedSchemaFile:        "testdata/mutations_expected.graphql",
			expectedFastgqlSchemaFile: "testdata/mutations_fastgql_expected.graphql",
		},
		{
			name:                      "mutations_with_filters",
			baseSchemaFile:            "testdata/mutations.graphql",
			expectedSchemaFile:        "testdata/mutations_filter_expected.graphql",
			expectedFastgqlSchemaFile: "testdata/mutations_fastgql_filter_expected.graphql",
			Augmenter:                 []Augmenter{FilterInputAugmenter, FilterArgAugmenter},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			generateTestRunner(t, &tc, MutationsAugmenter)
		})
	}
}

// Test_MutationsAugmenter_Unit tests the MutationsAugmenter function in isolation
func Test_MutationsAugmenter_Unit(t *testing.T) {
	tests := []struct {
		name             string
		schemaDefinition string
		expectCreate     bool
		expectUpdate     bool
		expectDelete     bool
		createMutation   string
		updateMutation   string
		deleteMutation   string
	}{
		{
			name: "creates_all_mutations_when_all_true",
			schemaDefinition: `
				type User @generateMutations(create: true, update: true, delete: true) {
					id: ID!
					name: String!
				}
			`,
			expectCreate:   true,
			expectUpdate:   true,
			expectDelete:   true,
			createMutation: "createUsers",
			updateMutation: "updateUsers",
			deleteMutation: "deleteUsers",
		},
		{
			name: "creates_only_specified_mutations",
			schemaDefinition: `
				type User @generateMutations(create: true, update: false, delete: false) {
					id: ID!
					name: String!
				}
			`,
			expectCreate:   true,
			expectUpdate:   false,
			expectDelete:   false,
			createMutation: "createUsers",
		},
		{
			name: "creates_update_and_delete_only",
			schemaDefinition: `
				type Post @generateMutations(create: false, update: true, delete: true) {
					id: ID!
					title: String!
				}
			`,
			expectCreate:   false,
			expectUpdate:   true,
			expectDelete:   true,
			updateMutation: "updatePosts",
			deleteMutation: "deletePosts",
		},
		{
			name: "skips_types_without_directive",
			schemaDefinition: `
				type User {
					id: ID!
					name: String!
				}
			`,
			expectCreate: false,
			expectUpdate: false,
			expectDelete: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := buildTestSchema(t, tt.schemaDefinition)

			err := MutationsAugmenter(schema)
			require.NoError(t, err)

			// Check if Mutation type was created
			if tt.expectCreate || tt.expectUpdate || tt.expectDelete {
				assert.NotNil(t, schema.Mutation, "Mutation type should be created")

				if tt.expectCreate {
					field := schema.Mutation.Fields.ForName(tt.createMutation)
					assert.NotNil(t, field, "Create mutation should exist")
				}
				if tt.expectUpdate {
					field := schema.Mutation.Fields.ForName(tt.updateMutation)
					assert.NotNil(t, field, "Update mutation should exist")
				}
				if tt.expectDelete {
					field := schema.Mutation.Fields.ForName(tt.deleteMutation)
					assert.NotNil(t, field, "Delete mutation should exist")
				}
			} else {
				// If no mutations expected and no Mutation type existed, it shouldn't be created
				if schema.Mutation != nil {
					assert.Empty(t, schema.Mutation.Fields, "No mutation fields should be added")
				}
			}
		})
	}
}

// Test_addCreateMutation tests create mutation generation
func Test_addCreateMutation(t *testing.T) {
	tests := []struct {
		name             string
		schemaDefinition string
		typeName         string
		expectedInput    string
		expectedFields   []string
		skipFields       []string
	}{
		{
			name: "creates_input_with_scalar_fields",
			schemaDefinition: `
				type User {
					id: ID!
					name: String!
					age: Int
				}
			`,
			typeName:       "User",
			expectedInput:  "CreateUserInput",
			expectedFields: []string{"id", "name", "age"},
		},
		{
			name: "skips_composite_fields",
			schemaDefinition: `
				type User {
					id: ID!
					name: String!
					posts: [Post]
				}
				type Post {
					id: ID!
				}
			`,
			typeName:       "User",
			expectedInput:  "CreateUserInput",
			expectedFields: []string{"id", "name"},
			skipFields:     []string{"posts"},
		},
		{
			name: "includes_all_scalar_fields",
			schemaDefinition: `
				type User {
					id: ID!
					_internal: String
					name: String!
				}
			`,
			typeName:       "User",
			expectedInput:  "CreateUserInput",
			expectedFields: []string{"id", "name", "_internal"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := buildTestSchema(t, tt.schemaDefinition)
			objDef := schema.Types[tt.typeName]
			require.NotNil(t, objDef)

			result := addCreateMutation(schema, objDef)
			require.NotNil(t, result, "Create mutation field should be returned")

			// Check mutation field properties
			assert.Equal(t, "create"+inflection.Plural(tt.typeName), result.Name)
			assert.NotNil(t, result.Arguments.ForName("inputs"), "inputs argument should exist")

			// Check input type was created (stored by mutation name)
			inputKey := "create" + inflection.Plural(tt.typeName)
			inputType, exists := schema.Types[inputKey]
			require.True(t, exists, "Input type should be created with key %s", inputKey)
			assert.Equal(t, ast.InputObject, inputType.Kind)
			assert.Equal(t, tt.expectedInput, inputType.Name)

			// Check expected fields
			for _, fieldName := range tt.expectedFields {
				field := inputType.Fields.ForName(fieldName)
				assert.NotNil(t, field, "Expected field %s in input", fieldName)
			}

			// Check skipped fields
			for _, fieldName := range tt.skipFields {
				field := inputType.Fields.ForName(fieldName)
				assert.Nil(t, field, "Should skip field %s", fieldName)
			}
		})
	}
}

// Test_addUpdateMutation tests update mutation generation
func Test_addUpdateMutation(t *testing.T) {
	tests := []struct {
		name             string
		schemaDefinition string
		typeName         string
		expectedInput    string
		checkNullable    bool
	}{
		{
			name: "creates_update_input_with_nullable_fields",
			schemaDefinition: `
				type User {
					id: ID!
					name: String!
					age: Int!
				}
			`,
			typeName:      "User",
			expectedInput: "UpdateUserInput",
			checkNullable: true,
		},
		{
			name: "skips_composite_fields",
			schemaDefinition: `
				type User {
					id: ID!
					name: String!
					posts: [Post]
				}
				type Post {
					id: ID!
				}
			`,
			typeName:      "User",
			expectedInput: "UpdateUserInput",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := buildTestSchema(t, tt.schemaDefinition)
			objDef := schema.Types[tt.typeName]
			require.NotNil(t, objDef)

			result := addUpdateMutation(schema, objDef)
			require.NotNil(t, result, "Update mutation field should be returned")

			// Check mutation field properties
			assert.Equal(t, "update"+inflection.Plural(tt.typeName), result.Name)
			assert.NotNil(t, result.Arguments.ForName("input"), "input argument should exist")

			// Check input type was created (stored by mutation name)
			inputKey := "update" + inflection.Plural(tt.typeName)
			inputType, exists := schema.Types[inputKey]
			require.True(t, exists, "Input type should be created with key %s", inputKey)
			assert.Equal(t, tt.expectedInput, inputType.Name)

			// Check fields are nullable
			if tt.checkNullable {
				for _, field := range inputType.Fields {
					assert.False(t, field.Type.NonNull, "Field %s should be nullable in update input", field.Name)
				}
			}
		})
	}
}

// Test_addDeleteMutation tests delete mutation generation
func Test_addDeleteMutation(t *testing.T) {
	tests := []struct {
		name              string
		schemaDefinition  string
		typeName          string
		expectCascade     bool
		expectFilterArg   bool
	}{
		{
			name: "creates_delete_with_cascade",
			schemaDefinition: `
				type User {
					id: ID!
					name: String!
				}
			`,
			typeName:      "User",
			expectCascade: true,
		},
		{
			name: "adds_filter_when_filter_input_exists",
			schemaDefinition: `
				type User @generateFilterInput {
					id: ID!
					name: String!
				}
			`,
			typeName:        "User",
			expectCascade:   true,
			expectFilterArg: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := buildTestSchema(t, tt.schemaDefinition)
			
			// Create filter inputs if needed
			if tt.expectFilterArg {
				_ = FilterInputAugmenter(schema)
			}

			objDef := schema.Types[tt.typeName]
			require.NotNil(t, objDef)

			result := addDeleteMutation(schema, objDef)
			require.NotNil(t, result, "Delete mutation field should be returned")

			// Check mutation field properties
			assert.Equal(t, "delete"+inflection.Plural(tt.typeName), result.Name)

			// Check cascade argument
			if tt.expectCascade {
				cascadeArg := result.Arguments.ForName("cascade")
				assert.NotNil(t, cascadeArg, "cascade argument should exist")
			}

			// Check filter argument
			if tt.expectFilterArg {
				filterArg := result.Arguments.ForName("filter")
				assert.NotNil(t, filterArg, "filter argument should exist when filter input exists")
			}
		})
	}
}

// Test_getPayloadObject tests payload object creation
func Test_getPayloadObject(t *testing.T) {
	tests := []struct {
		name              string
		schemaDefinition  string
		typeName          string
		expectedPayload   string
		alreadyExists     bool
	}{
		{
			name: "creates_new_payload_object",
			schemaDefinition: `
				type User {
					id: ID!
					name: String!
				}
			`,
			typeName:        "User",
			expectedPayload: "UsersPayload",
			alreadyExists:   false,
		},
		{
			name: "returns_existing_payload",
			schemaDefinition: `
				type User {
					id: ID!
					name: String!
				}
				type UsersPayload {
					rows_affected: Int!
					users: [User]
				}
			`,
			typeName:        "User",
			expectedPayload: "UsersPayload",
			alreadyExists:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := buildTestSchema(t, tt.schemaDefinition)
			objDef := schema.Types[tt.typeName]
			require.NotNil(t, objDef)

			result := getPayloadObject(schema, objDef)
			require.NotNil(t, result, "Payload object should be returned")
			assert.Equal(t, tt.expectedPayload, result.Name)

			// Check payload fields
			assert.NotNil(t, result.Fields.ForName("rows_affected"), "rows_affected field should exist")
			assert.NotNil(t, result.Fields.ForName(inflection.Plural(strcase.ToSnake(tt.typeName))), "type field should exist")

			// Check it's in schema
			payloadInSchema, exists := schema.Types[tt.expectedPayload]
			assert.True(t, exists, "Payload should be in schema")
			assert.Equal(t, result, payloadInSchema, "Should be same object")
		})
	}
}

// Test_schemaHasMutationDirective tests directive checking
func Test_schemaHasMutationDirective(t *testing.T) {
	tests := []struct {
		name             string
		schemaDefinition string
		expected         bool
	}{
		{
			name: "returns_true_when_directive_exists",
			schemaDefinition: `
				type User @generateMutations(create: true, update: true, delete: true) {
					id: ID!
				}
			`,
			expected: true,
		},
		{
			name: "returns_false_when_no_directive",
			schemaDefinition: `
				type User {
					id: ID!
				}
				type Post {
					id: ID!
				}
			`,
			expected: false,
		},
		{
			name: "returns_true_when_any_type_has_directive",
			schemaDefinition: `
				type User {
					id: ID!
				}
				type Post @generateMutations(create: true) {
					id: ID!
				}
			`,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := buildTestSchema(t, tt.schemaDefinition)
			result := schemaHasMutationDirective(schema)
			assert.Equal(t, tt.expected, result)
		})
	}
}

