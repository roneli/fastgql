package schema

import (
	"testing"
)

func Test_FilterInput(t *testing.T) {
	var testCases = []generateTestCase{
		{
			name:                      "filter_base",
			baseSchemaFile:            "testdata/base.graphql",
			expectedSchemaFile:        "testdata/base_filter_only_expected.graphql",
			expectedFastgqlSchemaFile: "testdata/base_filter_only_fastgql_expected.graphql",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			generateTestRunner(t, &tc, FilterInputAugment, FilterArgumentsAugment)
		})
	}
}
