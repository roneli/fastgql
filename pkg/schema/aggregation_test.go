package schema

import (
	"testing"
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
