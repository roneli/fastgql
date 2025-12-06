package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_AugmentPagination(t *testing.T) {
	var testCases = []generateTestCase{
		{
			name:               "pagination_base",
			baseSchemaFile:     "testdata/base.graphql",
			expectedSchemaFile: "testdata/pagination_expected.graphql",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			generateTestRunner(t, &tc, PaginationAugmenter)
		})
	}
}

// Test_PaginationAugmenter_Unit tests the PaginationAugmenter function in isolation
func Test_PaginationAugmenter_Unit(t *testing.T) {
	tests := []struct {
		name             string
		schemaDefinition string
		queryField       string
		expectPagination bool
	}{
		{
			name: "adds_pagination_to_list_field_with_directive",
			schemaDefinition: `
				type User {
					id: ID!
					name: String!
				}
				type Query {
					users: [User] @generate(pagination: true)
				}
			`,
			queryField:       "users",
			expectPagination: true,
		},
		{
			name: "skips_pagination_when_directive_false",
			schemaDefinition: `
				type User {
					id: ID!
				}
				type Query {
					users: [User] @generate(pagination: false)
				}
			`,
			queryField:       "users",
			expectPagination: false,
		},
		{
			name: "skips_non_list_fields",
			schemaDefinition: `
				type User {
					id: ID!
				}
				type Query {
					user: User @generate(pagination: true)
				}
			`,
			queryField:       "user",
			expectPagination: false,
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
			queryField:       "users",
			expectPagination: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := buildTestSchema(t, tt.schemaDefinition)

			err := PaginationAugmenter(schema)
			require.NoError(t, err)

			field := schema.Query.Fields.ForName(tt.queryField)
			require.NotNil(t, field, "Query field should exist")

			limitArg := field.Arguments.ForName("limit")
			offsetArg := field.Arguments.ForName("offset")

			if tt.expectPagination {
				assert.NotNil(t, limitArg, "limit argument should exist")
				assert.NotNil(t, offsetArg, "offset argument should exist")

				// Check default values
				if limitArg != nil && limitArg.DefaultValue != nil {
					assert.Equal(t, "100", limitArg.DefaultValue.Raw, "limit default should be 100")
				}
				if offsetArg != nil && offsetArg.DefaultValue != nil {
					assert.Equal(t, "0", offsetArg.DefaultValue.Raw, "offset default should be 0")
				}
			} else {
				assert.Nil(t, limitArg, "limit argument should not exist")
				assert.Nil(t, offsetArg, "offset argument should not exist")
			}
		})
	}
}

// Test_addPaginationToField tests adding pagination arguments to fields
func Test_addPaginationToField(t *testing.T) {
	tests := []struct {
		name             string
		schemaDefinition string
		objectName       string
		fieldName        string
		shouldAdd        bool
	}{
		{
			name: "adds_pagination_to_field",
			schemaDefinition: `
				type User {
					id: ID!
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
			name: "skips_when_limit_already_exists",
			schemaDefinition: `
				type User {
					id: ID!
				}
				type Query {
					users(limit: Int): [User]
				}
			`,
			objectName: "Query",
			fieldName:  "users",
			shouldAdd:  false,
		},
		{
			name: "skips_when_offset_already_exists",
			schemaDefinition: `
				type User {
					id: ID!
				}
				type Query {
					users(offset: Int): [User]
				}
			`,
			objectName: "Query",
			fieldName:  "users",
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

			err := addPaginationToField(schema, obj, field)
			require.NoError(t, err)

			limitArg := field.Arguments.ForName("limit")
			offsetArg := field.Arguments.ForName("offset")

			if tt.shouldAdd {
				assert.NotNil(t, limitArg, "limit argument should be added")
				assert.NotNil(t, offsetArg, "offset argument should be added")
				assert.Equal(t, initialArgCount+2, len(field.Arguments), "Should add 2 arguments")
			} else {
				// If already exists, count should not increase
				assert.Equal(t, initialArgCount, len(field.Arguments), "Argument count should not change")
			}
		})
	}
}
