package schema

import (
	"testing"
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
