package schema

import (
	"testing"
)

func Test_AugmentMutations(t *testing.T) {
	var testCases = []generateTestCase{
		{
			name:                      "ordering_base",
			baseSchemaFile:            "testdata/mutations.graphql",
			expectedSchemaFile:        "testdata/mutations_expected.graphql",
			expectedFastgqlSchemaFile: "testdata/mutations_fastgql_expected.graphql",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			generateTestRunner(t, &tc, MutationsAugmenter)
		})
	}
}
