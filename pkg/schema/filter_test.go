package schema

import (
	"testing"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
)

// Test_FilterInput tests the complete filter generation pipeline
func Test_FilterInput(t *testing.T) {
	var testCases = []generateTestCase{
		{
			name:                      "filter_base",
			baseSchemaFile:            "testdata/base.graphql",
			expectedSchemaFile:        "testdata/base_filter_only_expected.graphql",
			expectedFastgqlSchemaFile: "testdata/base_filter_only_fastgql_expected.graphql",
		},
		{
			name:                      "filter_interface",
			baseSchemaFile:            "testdata/filter_interface.graphql",
			expectedSchemaFile:        "testdata/filter_interface_expected.graphql",
			expectedFastgqlSchemaFile: "testdata/filter_interface_fastgql_expected.graphql",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			generateTestRunner(t, &tc, FilterInputAugmenter, FilterArgAugmenter)
		})
	}
}

// Test_FilterInputAugmenter_Unit tests the FilterInputAugmenter function in isolation
func Test_FilterInputAugmenter_Unit(t *testing.T) {
	tests := []struct {
		name                string
		schemaDefinition    string
		expectedInputNames  []string
		expectLogicalOps    bool
		expectInterfaceImpl bool
	}{
		{
			name: "creates_filter_input_for_types_with_directive",
			schemaDefinition: `
				type User @generateFilterInput {
					id: ID!
					name: String!
				}
				type Query {
					users: [User]
				}
			`,
			expectedInputNames: []string{"UserFilterInput"},
			expectLogicalOps:   true,
		},
		{
			name: "skips_types_without_directive",
			schemaDefinition: `
				type User {
					id: ID!
					name: String!
				}
				type Query {
					users: [User]
				}
			`,
			expectedInputNames: []string{},
		},
		{
			name: "handles_interface_types",
			schemaDefinition: `
				interface Animal @generateFilterInput {
					id: ID!
					name: String!
				}
				type Cat implements Animal {
					id: ID!
					name: String!
					meow: String
				}
				type Dog implements Animal {
					id: ID!
					name: String!
					bark: String
				}
				type Query {
					animals: [Animal]
				}
			`,
			expectedInputNames:  []string{"AnimalFilterInput", "CatFilterInput", "DogFilterInput"},
			expectInterfaceImpl: true,
		},
		{
			name: "handles_multiple_types",
			schemaDefinition: `
				type User @generateFilterInput {
					id: ID!
				}
				type Post @generateFilterInput {
					id: ID!
				}
				type Query {
					users: [User]
					posts: [Post]
				}
			`,
			expectedInputNames: []string{"UserFilterInput", "PostFilterInput"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := buildTestSchema(t, tt.schemaDefinition)

			// Run the augmenter
			err := FilterInputAugmenter(schema)
			require.NoError(t, err)

			// Verify expected filter inputs were created
			for _, inputName := range tt.expectedInputNames {
				input, exists := schema.Types[inputName]
				assert.True(t, exists, "Expected filter input %s to exist", inputName)
				if exists {
					assert.Equal(t, ast.InputObject, input.Kind)

					// Check for logical operators
					if tt.expectLogicalOps {
						assert.NotNil(t, input.Fields.ForName("AND"), "Expected AND operator")
						assert.NotNil(t, input.Fields.ForName("OR"), "Expected OR operator")
						assert.NotNil(t, input.Fields.ForName("NOT"), "Expected NOT operator")
					}
				}
			}

			// Verify no unexpected inputs were created
			if len(tt.expectedInputNames) == 0 {
				for name, typeDef := range schema.Types {
					if typeDef.Kind == ast.InputObject {
						assert.NotContains(t, name, "FilterInput", "Unexpected filter input created: %s", name)
					}
				}
			}
		})
	}
}

// Test_FilterArgAugmenter_Unit tests the FilterArgAugmenter function in isolation
func Test_FilterArgAugmenter_Unit(t *testing.T) {
	tests := []struct {
		name             string
		schemaDefinition string
		fieldName        string
		expectFilterArg  bool
		expectError      bool
	}{
		{
			name: "adds_filter_to_query_field_with_directive",
			schemaDefinition: `
				type User @generateFilterInput {
					id: ID!
					name: String!
				}
				type Query {
					users: [User] @generate(filter: true)
				}
			`,
			fieldName:       "users",
			expectFilterArg: true,
		},
		{
			name: "skips_field_without_directive",
			schemaDefinition: `
				type User @generateFilterInput {
					id: ID!
				}
				type Query {
					users: [User]
				}
			`,
			fieldName:       "users",
			expectFilterArg: false,
		},
		{
			name: "skips_field_with_filter_false",
			schemaDefinition: `
				type User @generateFilterInput {
					id: ID!
				}
				type Query {
					users: [User] @generate(filter: false)
				}
			`,
			fieldName:       "users",
			expectFilterArg: false,
		},
		{
			name: "handles_aggregate_fields",
			schemaDefinition: `
				type User @generateFilterInput {
					id: ID!
					name: String!
				}
				type Query {
					users: [User]
					_usersAggregate: [UserAggregate] @generate(filter: true)
				}
				type UserAggregate {
					count: Int!
				}
			`,
			fieldName:       "_usersAggregate",
			expectFilterArg: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := buildTestSchema(t, tt.schemaDefinition)

			// First run FilterInputAugmenter to create the inputs
			err := FilterInputAugmenter(schema)
			require.NoError(t, err)

			// Then run FilterArgAugmenter
			err = FilterArgAugmenter(schema)
			if tt.expectError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Check if filter argument was added
			field := schema.Query.Fields.ForName(tt.fieldName)
			require.NotNil(t, field, "Field %s should exist", tt.fieldName)

			filterArg := field.Arguments.ForName("filter")
			if tt.expectFilterArg {
				assert.NotNil(t, filterArg, "Expected filter argument on field %s", tt.fieldName)
				if filterArg != nil {
					assert.Contains(t, filterArg.Type.Name(), "FilterInput")
				}
			} else {
				assert.Nil(t, filterArg, "Did not expect filter argument on field %s", tt.fieldName)
			}
		})
	}
}

// Test_buildFilterInput tests the filter input field generation
func Test_buildFilterInput(t *testing.T) {
	tests := []struct {
		name             string
		schemaDefinition string
		typeName         string
		expectedFields   []string
	}{
		{
			name: "always_includes_logical_operators",
			schemaDefinition: `
				type User @generateFilterInput {
					id: ID!
					name: String!
				}
				type Query {
					users: [User]
				}
			`,
			typeName: "User",
			expectedFields: []string{
				"AND", // Logical operators should always be present
				"OR",
				"NOT",
			},
		},
		{
			name: "creates_filters_for_relation_fields",
			schemaDefinition: `
				type User @generateFilterInput {
					id: ID!
					posts: [Post]
				}
				type Post @generateFilterInput {
					id: ID!
					title: String!
				}
				type Query {
					users: [User]
					posts: [Post]
				}
			`,
			typeName: "User",
			expectedFields: []string{
				"posts", // Relation filter inputs should be created
				"AND",
				"OR",
				"NOT",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := buildTestSchema(t, tt.schemaDefinition)

			// Run FilterInputAugmenter which calls buildFilterInput internally
			err := FilterInputAugmenter(schema)
			require.NoError(t, err)

			// Get the generated filter input
			filterInputName := tt.typeName + "FilterInput"
			filterInput, exists := schema.Types[filterInputName]
			require.True(t, exists, "Filter input %s should exist", filterInputName)

			// Check expected fields
			for _, fieldName := range tt.expectedFields {
				field := filterInput.Fields.ForName(fieldName)
				assert.NotNil(t, field, "Expected field %s to exist in %s", fieldName, filterInputName)
			}
		})
	}
}

// Test_initInputs tests the filter input initialization
func Test_initInputs(t *testing.T) {
	tests := []struct {
		name             string
		schemaDefinition string
		expectedInputs   []string
	}{
		{
			name: "initializes_filter_inputs",
			schemaDefinition: `
				type User @generateFilterInput {
					id: ID!
				}
				type Post @generateFilterInput(description: "Filter for posts") {
					id: ID!
				}
			`,
			expectedInputs: []string{"UserFilterInput", "PostFilterInput"},
		},
		{
			name: "initializes_interface_implementation_inputs",
			schemaDefinition: `
				interface Animal @generateFilterInput {
					id: ID!
				}
				type Cat implements Animal {
					id: ID!
					meow: String
				}
				type Dog implements Animal {
					id: ID!
					bark: String
				}
			`,
			expectedInputs: []string{"AnimalFilterInput", "CatFilterInput", "DogFilterInput"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := buildTestSchema(t, tt.schemaDefinition)

			// Call initInputs
			inputs := initInputs(schema)

			// Verify all expected inputs are in the schema
			for _, inputName := range tt.expectedInputs {
				found := false
				for _, input := range inputs {
					if input.input.Name == inputName {
						found = true
						assert.Equal(t, ast.InputObject, input.input.Kind)
						break
					}
				}
				assert.True(t, found, "Expected input %s to be initialized", inputName)

				// Also check it's in the schema
				_, exists := schema.Types[inputName]
				assert.True(t, exists, "Input %s should be in schema", inputName)
			}
		})
	}
}

// Test_addFilterToQueryFieldArgs tests adding filter arguments to query fields
func Test_addFilterToQueryFieldArgs(t *testing.T) {
	tests := []struct {
		name             string
		schemaDefinition string
		fieldName        string
		shouldAddFilter  bool
		skipReason       string
	}{
		{
			name: "adds_filter_to_field",
			schemaDefinition: `
				type User @generateFilterInput {
					id: ID!
				}
				type Query {
					users: [User]
				}
			`,
			fieldName:       "users",
			shouldAddFilter: true,
		},
		{
			name: "skips_when_filter_already_exists",
			schemaDefinition: `
				type User @generateFilterInput {
					id: ID!
				}
				type Query {
					users(filter: String): [User]
				}
			`,
			fieldName:       "users",
			shouldAddFilter: false,
			skipReason:      "filter already exists",
		},
		{
			name: "handles_aggregate_field_name_transformation",
			schemaDefinition: `
				type User @generateFilterInput {
					id: ID!
				}
				type Query {
					users: [User]
					_usersAggregate: [UserAggregate]
				}
				type UserAggregate {
					count: Int!
				}
			`,
			fieldName:       "_usersAggregate",
			shouldAddFilter: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := buildTestSchema(t, tt.schemaDefinition)

			// Create filter inputs first
			err := FilterInputAugmenter(schema)
			require.NoError(t, err)

			field := schema.Query.Fields.ForName(tt.fieldName)
			require.NotNil(t, field)

			// Record initial arg count
			initialArgCount := len(field.Arguments)

			// Call the function
			err = addFilterToQueryFieldArgs(schema, schema.Query, field)
			require.NoError(t, err)

			// Check if filter was added
			filterArg := field.Arguments.ForName("filter")
			if tt.shouldAddFilter {
				if initialArgCount == len(field.Arguments) {
					// Filter already existed
					assert.NotNil(t, filterArg, "Filter argument should exist")
				} else {
					// Filter was added
					assert.NotNil(t, filterArg, "Filter argument should be added")
					assert.Equal(t, initialArgCount+1, len(field.Arguments))
				}
			}
		})
	}
}

// Test_addFilterToMutationField tests adding filter to mutation fields
func Test_addFilterToMutationField(t *testing.T) {
	tests := []struct {
		name             string
		schemaDefinition string
		mutationField    string
		filterTypeName   string
		expectFilter     bool
	}{
		{
			name: "adds_filter_to_update_mutation",
			schemaDefinition: `
				type User @generateFilterInput {
					id: ID!
					name: String!
				}
				type UserPayload {
					user: User
				}
				type Mutation {
					updateUser(input: UpdateUserInput!): UserPayload
				}
				input UpdateUserInput {
					name: String
				}
			`,
			mutationField:  "updateUser",
			filterTypeName: "UserFilterInput",
			expectFilter:   true,
		},
		{
			name: "adds_filter_to_delete_mutation",
			schemaDefinition: `
				type User @generateFilterInput {
					id: ID!
				}
				type UserPayload {
					user: User
				}
				type Mutation {
					deleteUser: UserPayload
				}
			`,
			mutationField:  "deleteUser",
			filterTypeName: "UserFilterInput",
			expectFilter:   true,
		},
		{
			name: "skips_when_filter_type_not_exists",
			schemaDefinition: `
				type User {
					id: ID!
				}
				type Mutation {
					deleteUser: User
				}
			`,
			mutationField:  "deleteUser",
			filterTypeName: "UserFilterInput",
			expectFilter:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := buildTestSchema(t, tt.schemaDefinition)

			// Create filter inputs
			_ = FilterInputAugmenter(schema)

			field := schema.Mutation.Fields.ForName(tt.mutationField)
			require.NotNil(t, field, "Mutation field should exist")

			// Call the function
			err := addFilterToMutationField(schema, field, tt.filterTypeName)
			require.NoError(t, err)

			// Check result
			filterArg := field.Arguments.ForName("filter")
			if tt.expectFilter {
				assert.NotNil(t, filterArg, "Expected filter argument")
				if filterArg != nil {
					assert.Equal(t, tt.filterTypeName, filterArg.Type.Name())
				}
			} else {
				assert.Nil(t, filterArg, "Did not expect filter argument")
			}
		})
	}
}

// Test_FilterInput_JsonTypes tests filter generation for JSON fields
func Test_FilterInput_JsonTypes(t *testing.T) {
	tests := []struct {
		name             string
		schemaDefinition string
		typeName         string
		expectedFilters  map[string]string // field name -> filter type name
	}{
		{
			name: "creates_filter_input_for_json_object_type",
			schemaDefinition: `
				type ProductAttributes {
					color: String!
					size: Int!
				}
				type Product @generateFilterInput {
					id: ID!
					name: String!
					attributes: ProductAttributes @json(column: "attributes")
				}
				type Query {
					products: [Product]
				}
			`,
			typeName: "Product",
			expectedFilters: map[string]string{
				"id":         "IDComparator",
				"name":       "StringComparator",
				"attributes": "ProductAttributesFilterInput",
			},
		},
		{
			name: "json_filter_input_uses_jsonpath_comparators",
			schemaDefinition: `
				type ProductAttributes {
					color: String!
					size: Int!
					isActive: Boolean!
				}
				type Product @generateFilterInput {
					id: ID!
					attributes: ProductAttributes @json(column: "attributes")
				}
				type Query {
					products: [Product]
				}
			`,
			typeName: "ProductAttributes",
			expectedFilters: map[string]string{
				"color":    "JsonPathStringComparator",
				"size":     "JsonPathIntComparator",
				"isActive": "JsonPathBooleanComparator",
			},
		},
		{
			name: "creates_nested_filter_inputs_for_nested_json_types",
			schemaDefinition: `
				type Address {
					street: String!
					city: String!
				}
				type UserProfile {
					bio: String!
					address: Address
				}
				type User @generateFilterInput {
					id: ID!
					profile: UserProfile @json(column: "profile")
				}
				type Query {
					users: [User]
				}
			`,
			typeName: "User",
			expectedFilters: map[string]string{
				"id":      "IDComparator",
				"profile": "UserProfileFilterInput",
			},
		},
		{
			name: "handles_json_directive_fields",
			schemaDefinition: `
				type ProductAttributes {
					color: String!
				}
				type Product @generateFilterInput {
					id: ID!
					name: String!
					attributes: ProductAttributes @json(column: "attributes")
				}
				type Query {
					products: [Product]
				}
			`,
			typeName: "Product",
			expectedFilters: map[string]string{
				"id":         "IDComparator",
				"name":       "StringComparator",
				"attributes": "ProductAttributesFilterInput",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := buildTestSchema(t, tt.schemaDefinition)

			// Run FilterInputAugmenter
			err := FilterInputAugmenter(schema)
			require.NoError(t, err)

			// Get the generated filter input
			filterInputName := tt.typeName + "FilterInput"
			filterInput, exists := schema.Types[filterInputName]
			require.True(t, exists, "Filter input %s should exist", filterInputName)

			// Check expected filter fields
			for fieldName, expectedFilterType := range tt.expectedFilters {
				field := filterInput.Fields.ForName(fieldName)
				assert.NotNil(t, field, "Expected field %s to exist in %s", fieldName, filterInputName)
				if field != nil {
					assert.Equal(t, expectedFilterType, field.Type.Name(),
						"Field %s should have filter type %s", fieldName, expectedFilterType)
				}
			}

			// Verify logical operators are present
			assert.NotNil(t, filterInput.Fields.ForName("AND"), "Expected AND operator")
			assert.NotNil(t, filterInput.Fields.ForName("OR"), "Expected OR operator")
			assert.NotNil(t, filterInput.Fields.ForName("NOT"), "Expected NOT operator")
		})
	}
}

// Test_createJsonTypeFilterInput tests the creation of filter inputs for JSON object types
func Test_createJsonTypeFilterInput(t *testing.T) {
	tests := []struct {
		name             string
		schemaDefinition string
		jsonTypeName     string
		expectedFields   []string
		checkLogicalOps  bool
	}{
		{
			name: "creates_filter_with_basic_scalar_fields",
			schemaDefinition: `
				type ProductAttributes {
					color: String!
					size: Int!
					available: Boolean!
				}
			`,
			jsonTypeName: "ProductAttributes",
			expectedFields: []string{
				"color",
				"size",
				"available",
				"AND",
				"OR",
				"NOT",
			},
			checkLogicalOps: true,
		},
		{
			name: "creates_filter_with_nested_object",
			schemaDefinition: `
				type Address {
					street: String!
					city: String!
				}
				type UserProfile {
					bio: String!
					address: Address
				}
			`,
			jsonTypeName: "UserProfile",
			expectedFields: []string{
				"bio",
				"address",
				"AND",
				"OR",
				"NOT",
			},
			checkLogicalOps: true,
		},
		{
			name: "handles_list_types",
			schemaDefinition: `
				type Tags {
					names: [String!]
					counts: [Int!]
				}
			`,
			jsonTypeName: "Tags",
			expectedFields: []string{
				"names",
				"counts",
				"AND",
				"OR",
				"NOT",
			},
			checkLogicalOps: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := buildTestSchema(t, tt.schemaDefinition)

			jsonType := schema.Types[tt.jsonTypeName]
			require.NotNil(t, jsonType, "JSON type %s should exist", tt.jsonTypeName)

			filterInputName := tt.jsonTypeName + "FilterInput"

			// Call createJsonTypeFilterInput
			createJsonTypeFilterInput(schema, jsonType, filterInputName)

			// Verify the filter input was created
			filterInput, exists := schema.Types[filterInputName]
			require.True(t, exists, "Filter input %s should be created", filterInputName)
			assert.Equal(t, ast.InputObject, filterInput.Kind)

			// Check expected fields
			for _, fieldName := range tt.expectedFields {
				field := filterInput.Fields.ForName(fieldName)
				assert.NotNil(t, field, "Expected field %s to exist in %s", fieldName, filterInputName)
			}

			// Verify logical operators if requested
			if tt.checkLogicalOps {
				andField := filterInput.Fields.ForName("AND")
				orField := filterInput.Fields.ForName("OR")
				notField := filterInput.Fields.ForName("NOT")

				assert.NotNil(t, andField, "AND operator should exist")
				assert.NotNil(t, orField, "OR operator should exist")
				assert.NotNil(t, notField, "NOT operator should exist")

				// Verify AND/OR are arrays and NOT is single
				if andField != nil {
					assert.NotNil(t, andField.Type.Elem, "AND should be an array type")
				}
				if orField != nil {
					assert.NotNil(t, orField.Type.Elem, "OR should be an array type")
				}
				if notField != nil {
					assert.Nil(t, notField.Type.Elem, "NOT should be a single type, not an array")
				}
			}
		})
	}
}

// Test_JsonFilter_NestedObjects tests deeply nested JSON object filter generation
func Test_JsonFilter_NestedObjects(t *testing.T) {
	schemaDefinition := `
		type Location {
			lat: Float!
			lng: Float!
		}
		type Address {
			street: String!
			location: Location
		}
		type UserProfile {
			bio: String!
			address: Address
		}
		type User @generateFilterInput {
			id: ID!
			profile: UserProfile @json(column: "profile")
		}
		type Query {
			users: [User]
		}
	`

	schema := buildTestSchema(t, schemaDefinition)

	// Run FilterInputAugmenter
	err := FilterInputAugmenter(schema)
	require.NoError(t, err)

	// Verify all nested filter inputs were created
	filterInputs := []string{
		"UserFilterInput",
		"UserProfileFilterInput",
		"AddressFilterInput",
		"LocationFilterInput",
	}

	for _, inputName := range filterInputs {
		filterInput, exists := schema.Types[inputName]
		assert.True(t, exists, "Filter input %s should exist", inputName)
		if exists {
			assert.Equal(t, ast.InputObject, filterInput.Kind)
			// Verify logical operators
			assert.NotNil(t, filterInput.Fields.ForName("AND"))
			assert.NotNil(t, filterInput.Fields.ForName("OR"))
			assert.NotNil(t, filterInput.Fields.ForName("NOT"))
		}
	}

	// Verify the field references are correct
	userFilter := schema.Types["UserFilterInput"]
	profileField := userFilter.Fields.ForName("profile")
	require.NotNil(t, profileField)
	assert.Equal(t, "UserProfileFilterInput", profileField.Type.Name())

	profileFilter := schema.Types["UserProfileFilterInput"]
	addressField := profileFilter.Fields.ForName("address")
	require.NotNil(t, addressField)
	assert.Equal(t, "AddressFilterInput", addressField.Type.Name())

	addressFilter := schema.Types["AddressFilterInput"]
	locationField := addressFilter.Fields.ForName("location")
	require.NotNil(t, locationField)
	assert.Equal(t, "LocationFilterInput", locationField.Type.Name())
}

// buildTestSchema is a helper to build a schema from a GraphQL SDL string
func buildTestSchema(t *testing.T, schemaSDL string) *ast.Schema {
	t.Helper()

	// Add fastgql types to the schema
	fullSchema := fastgql + "\n" + schemaSDL

	cfg := config.DefaultConfig()
	cfg.Sources = []*ast.Source{
		{
			Name:    "test.graphql",
			Input:   fullSchema,
			BuiltIn: false,
		},
	}

	err := cfg.LoadSchema()
	require.NoError(t, err, "Failed to load test schema")

	return cfg.Schema
}
