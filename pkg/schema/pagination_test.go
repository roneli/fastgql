package schema

import (
	"testing"
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
