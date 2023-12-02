package schema

import (
	"testing"
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
