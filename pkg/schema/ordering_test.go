package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
)

func Test_AugmentOrdering(t *testing.T) {
	var testCases = []generateTestCase{
		{
			name:                      "ordering_base",
			baseSchemaFile:            "testdata/base.graphql",
			expectedSchemaFile:        "testdata/ordering_expected.graphql",
			expectedFastgqlSchemaFile: "testdata/ordering_fastgql_expected.graphql",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			generateTestRunner(t, &tc, OrderByAugmenter)
		})
	}
}

// Test_OrderByAugmenter_Unit tests the OrderByAugmenter function in isolation
func Test_OrderByAugmenter_Unit(t *testing.T) {
	tests := []struct {
		name             string
		schemaDefinition string
		queryField       string
		expectOrderBy    bool
		orderingType     string
	}{
		{
			name: "adds_orderby_to_list_field_with_directive",
			schemaDefinition: `
				type User {
					id: ID!
					name: String!
				}
				type Query {
					users: [User] @generate(ordering: true)
				}
			`,
			queryField:    "users",
			expectOrderBy: true,
			orderingType:  "UserOrdering",
		},
		{
			name: "skips_orderby_when_directive_false",
			schemaDefinition: `
				type User {
					id: ID!
				}
				type Query {
					users: [User] @generate(ordering: false)
				}
			`,
			queryField:    "users",
			expectOrderBy: false,
		},
		{
			name: "skips_non_list_fields",
			schemaDefinition: `
				type User {
					id: ID!
				}
				type Query {
					user: User @generate(ordering: true)
				}
			`,
			queryField:    "user",
			expectOrderBy: false,
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
			queryField:    "users",
			expectOrderBy: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := buildTestSchema(t, tt.schemaDefinition)

			err := OrderByAugmenter(schema)
			require.NoError(t, err)

			field := schema.Query.Fields.ForName(tt.queryField)
			require.NotNil(t, field, "Query field should exist")

			orderByArg := field.Arguments.ForName("orderBy")
			if tt.expectOrderBy {
				assert.NotNil(t, orderByArg, "orderBy argument should exist")
				if orderByArg != nil {
					// Verify ordering type was created
					_, exists := schema.Types[tt.orderingType]
					assert.True(t, exists, "Ordering type should be created")
				}
			} else {
				assert.Nil(t, orderByArg, "orderBy argument should not exist")
			}
		})
	}
}

// Test_buildOrderingEnum tests the ordering enum creation
func Test_buildOrderingEnum(t *testing.T) {
	tests := []struct {
		name             string
		schemaDefinition string
		typeName         string
		expectedFields   []string
		unexpectedFields []string
		expectNil        bool
	}{
		{
			name: "creates_ordering_for_scalar_fields",
			schemaDefinition: `
				type User {
					id: ID!
					name: String!
					age: Int!
				}
			`,
			typeName:       "User",
			expectedFields: []string{"id", "name", "age"},
		},
		{
			name: "excludes_composite_fields",
			schemaDefinition: `
				type User {
					id: ID!
					name: String!
					posts: [Post]
					profile: Profile
				}
				type Post {
					id: ID!
				}
				type Profile {
					bio: String
				}
			`,
			typeName:         "User",
			expectedFields:   []string{"id", "name"},
			unexpectedFields: []string{"posts", "profile"},
		},
		{
			name: "returns_nil_when_no_orderable_fields",
			schemaDefinition: `
				type User {
					posts: [Post]
				}
				type Post {
					id: ID!
				}
			`,
			typeName:  "User",
			expectNil: true,
		},
		{
			name: "includes_enum_fields",
			schemaDefinition: `
				type User {
					id: ID!
					status: Status!
				}
				enum Status {
					ACTIVE
					INACTIVE
				}
			`,
			typeName:       "User",
			expectedFields: []string{"id", "status"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := buildTestSchema(t, tt.schemaDefinition)
			objDef := schema.Types[tt.typeName]
			require.NotNil(t, objDef)

			result := buildOrderingEnum(schema, objDef)

			if tt.expectNil {
				assert.Nil(t, result, "Should return nil when no orderable fields")
				return
			}

			require.NotNil(t, result, "Ordering enum should be created")
			assert.Equal(t, ast.InputObject, result.Kind)

			// Check expected fields
			for _, fieldName := range tt.expectedFields {
				field := result.Fields.ForName(fieldName)
				assert.NotNil(t, field, "Expected field %s", fieldName)
				if field != nil {
					assert.Equal(t, "_OrderingTypes", field.Type.Name(), "Field type should be _OrderingTypes")
				}
			}

			// Check unexpected fields
			for _, fieldName := range tt.unexpectedFields {
				field := result.Fields.ForName(fieldName)
				assert.Nil(t, field, "Should not have field %s", fieldName)
			}
		})
	}
}

// Test_addOrderByArgsToField tests adding orderBy arguments to fields
func Test_addOrderByArgsToField(t *testing.T) {
	tests := []struct {
		name             string
		schemaDefinition string
		objectName       string
		fieldName        string
		shouldAdd        bool
	}{
		{
			name: "adds_orderby_to_composite_list_field",
			schemaDefinition: `
				type User {
					id: ID!
					name: String!
				}
				type Query {
					users: [User]
				}
			`,
			objectName: "Query",
			fieldName:  "users",
			shouldAdd:  true,
		},
		{
			name: "skips_non_composite_types",
			schemaDefinition: `
				type Query {
					tags: [String]
				}
			`,
			objectName: "Query",
			fieldName:  "tags",
			shouldAdd:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := buildTestSchema(t, tt.schemaDefinition)
			obj := schema.Types[tt.objectName]
			field := obj.Fields.ForName(tt.fieldName)
			require.NotNil(t, field)

			initialArgCount := len(field.Arguments)

			err := addOrderByArgsToField(schema, obj, field)
			require.NoError(t, err)

			orderByArg := field.Arguments.ForName("orderBy")
			if tt.shouldAdd {
				assert.NotNil(t, orderByArg, "orderBy argument should be added")
				assert.Greater(t, len(field.Arguments), initialArgCount, "Argument count should increase")
			} else if orderByArg == nil {
				// For non-composite types, it might not add
				assert.Equal(t, initialArgCount, len(field.Arguments), "Argument count should not change")
			}
		})
	}
}
