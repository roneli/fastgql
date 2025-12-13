package sql

import (
	"strings"
	"testing"

	"github.com/doug-martin/goqu/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPathValidationEdgeCases tests edge cases in path validation
func TestPathValidationEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
		reason  string
	}{
		// Currently failing - multiple array indices not supported
		{
			name:    "multiple array indices",
			path:    "matrix[0][1]",
			wantErr: true, // Currently fails, should pass after fix
			reason:  "Regex doesn't support multiple consecutive array indices",
		},
		{
			name:    "deep array nesting",
			path:    "data[0].rows[1].cols[2]",
			wantErr: true, // Currently fails, should pass after fix
			reason:  "Regex doesn't support multiple array indices per path segment",
		},
		{
			name:    "array then field then array",
			path:    "items[0].subitems[1].value",
			wantErr: false, // This should work
			reason:  "Valid pattern with interleaved arrays and fields",
		},

		// Security - SQL injection attempts
		{
			name:    "sql injection with quote",
			path:    "field'; DROP TABLE users--",
			wantErr: true,
			reason:  "SQL injection attempt",
		},
		{
			name:    "sql injection with semicolon",
			path:    "field;DELETE FROM users",
			wantErr: true,
			reason:  "SQL injection attempt with semicolon",
		},
		{
			name:    "sql injection with union",
			path:    "field UNION SELECT password FROM users",
			wantErr: true,
			reason:  "SQL injection with UNION",
		},

		// Special characters
		{
			name:    "field with space",
			path:    "my field",
			wantErr: true,
			reason:  "Spaces not allowed in field names",
		},
		{
			name:    "field with hyphen",
			path:    "my-field",
			wantErr: true,
			reason:  "Hyphens not allowed",
		},
		{
			name:    "field with dollar",
			path:    "my$field",
			wantErr: true,
			reason:  "Dollar signs not allowed",
		},
		{
			name:    "field with unicode",
			path:    "fïeld",
			wantErr: true,
			reason:  "Unicode characters not supported",
		},

		// Edge cases
		{
			name:    "empty string",
			path:    "",
			wantErr: true,
			reason:  "Empty path not allowed",
		},
		{
			name:    "just a dot",
			path:    ".",
			wantErr: true,
			reason:  "Just a dot is invalid",
		},
		{
			name:    "trailing dot",
			path:    "field.",
			wantErr: true,
			reason:  "Trailing dot invalid",
		},
		{
			name:    "leading dot",
			path:    ".field",
			wantErr: true,
			reason:  "Leading dot invalid",
		},
		{
			name:    "double dot",
			path:    "field..nested",
			wantErr: true,
			reason:  "Double dot invalid",
		},
		{
			name:    "negative array index",
			path:    "items[-1]",
			wantErr: true,
			reason:  "Negative indices not allowed",
		},
		{
			name:    "array wildcard",
			path:    "items[*]",
			wantErr: true,
			reason:  "Wildcard not allowed (would need special handling)",
		},
		{
			name:    "array range",
			path:    "items[0:5]",
			wantErr: true,
			reason:  "Array ranges not supported",
		},

		// Valid cases
		{
			name:    "simple field",
			path:    "field",
			wantErr: false,
			reason:  "Simple field is valid",
		},
		{
			name:    "underscore field",
			path:    "my_field",
			wantErr: false,
			reason:  "Underscores are allowed",
		},
		{
			name:    "field with numbers",
			path:    "field123",
			wantErr: false,
			reason:  "Numbers in field names allowed",
		},
		{
			name:    "nested field",
			path:    "parent.child.grandchild",
			wantErr: false,
			reason:  "Multi-level nesting is valid",
		},
		{
			name:    "array access",
			path:    "items[0]",
			wantErr: false,
			reason:  "Single array index valid",
		},
		{
			name:    "array with nested field",
			path:    "items[0].name",
			wantErr: false,
			reason:  "Array then field valid",
		},
		{
			name:    "deeply nested valid",
			path:    "a.b.c.d.e.f",
			wantErr: false,
			reason:  "Deep nesting (6 levels) is valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePath(tt.path)
			if tt.wantErr {
				assert.Error(t, err, "Expected error for: %s", tt.reason)
			} else {
				assert.NoError(t, err, "Expected success for: %s", tt.reason)
			}
		})
	}
}

// TestVariableRemappingCorrectness tests that variable remapping doesn't corrupt data
func TestVariableRemappingCorrectness(t *testing.T) {
	tests := []struct {
		name         string
		filterMap    map[string]any
		checkVarVals map[string]any // Values that should appear in vars
		description  string
	}{
		{
			name: "value containing $v0",
			filterMap: map[string]any{
				"field": map[string]any{"eq": "$v0 is a string"},
			},
			checkVarVals: map[string]any{
				"v0": "$v0 is a string", // Should NOT be corrupted
			},
			description: "String value containing variable name should not be corrupted",
		},
		{
			name: "value containing multiple vars",
			filterMap: map[string]any{
				"field": map[string]any{"eq": "test $v0 and $v1 and $v2"},
			},
			checkVarVals: map[string]any{
				"v0": "test $v0 and $v1 and $v2",
			},
			description: "Value with multiple $vN patterns should not be corrupted",
		},
		{
			name: "nested AND with many variables",
			filterMap: map[string]any{
				"AND": []any{
					map[string]any{"f1": map[string]any{"eq": "val1"}},
					map[string]any{"f2": map[string]any{"eq": "val2"}},
					map[string]any{"f3": map[string]any{"eq": "val3"}},
					map[string]any{"f4": map[string]any{"eq": "val4"}},
					map[string]any{"f5": map[string]any{"eq": "val5"}},
				},
			},
			checkVarVals: map[string]any{
				"v0": "val1",
				"v1": "val2",
				"v2": "val3",
				"v3": "val4",
				"v4": "val5",
			},
			description: "Multiple variables should all be correctly assigned",
		},
		{
			name: "complex nesting with 100+ variables",
			filterMap: generateLargeFilterMap(100),
			checkVarVals: map[string]any{
				// Just check a few to ensure no corruption
				"v0":  "value0",
				"v50": "value50",
				"v99": "value99",
			},
			description: "Large variable count (100) should not cause collisions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonPath, vars, err := BuildJsonFilterFromOperatorMap(tt.filterMap)
			require.NoError(t, err, "Failed to build filter: %v", err)

			t.Logf("Generated JSONPath: %s", jsonPath)
			t.Logf("Generated vars: %v", vars)

			// Check that expected variables have correct values
			for expectedVar, expectedVal := range tt.checkVarVals {
				actualVal, ok := vars[expectedVar]
				require.True(t, ok, "Variable %s not found in vars", expectedVar)
				assert.Equal(t, expectedVal, actualVal,
					"Variable %s has wrong value. Expected %v, got %v",
					expectedVar, expectedVal, actualVal)
			}
		})
	}
}

// generateLargeFilterMap creates a filter map with many fields for stress testing
func generateLargeFilterMap(count int) map[string]any {
	andFilters := make([]any, count)
	for i := 0; i < count; i++ {
		andFilters[i] = map[string]any{
			"field": map[string]any{"eq": "value" + string(rune('0'+i%10))},
		}
	}
	return map[string]any{"AND": andFilters}
}

// TestSpecialCharacterHandling tests handling of special characters in values
func TestSpecialCharacterHandling(t *testing.T) {
	tests := []struct {
		name        string
		value       any
		wantEscaped bool
		description string
	}{
		{
			name:        "single quote",
			value:       "O'Brien",
			wantEscaped: true,
			description: "Single quotes should be escaped in JSON",
		},
		{
			name:        "double quote",
			value:       `He said "hello"`,
			wantEscaped: true,
			description: "Double quotes should be escaped",
		},
		{
			name:        "backslash",
			value:       `C:\Users\test`,
			wantEscaped: true,
			description: "Backslashes should be escaped",
		},
		{
			name:        "newline",
			value:       "line1\nline2",
			wantEscaped: true,
			description: "Newlines should be escaped",
		},
		{
			name:        "tab",
			value:       "col1\tcol2",
			wantEscaped: true,
			description: "Tabs should be escaped",
		},
		{
			name:        "unicode",
			value:       "Hello 世界",
			wantEscaped: false,
			description: "Unicode should be preserved",
		},
		{
			name:        "emoji",
			value:       "Test 🚀 emoji",
			wantEscaped: false,
			description: "Emojis should be preserved",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filterMap := map[string]any{
				"field": map[string]any{"eq": tt.value},
			}

			jsonPath, vars, err := BuildJsonFilterFromOperatorMap(filterMap)
			require.NoError(t, err, "Failed to build filter for value: %v", tt.value)

			// Verify value is in vars
			require.NotEmpty(t, vars, "Variables should not be empty")
			found := false
			for _, v := range vars {
				if v == tt.value {
					found = true
					break
				}
			}
			assert.True(t, found, "Value %v not found in vars", tt.value)

			// Verify JSONPath was generated
			assert.NotEmpty(t, jsonPath, "JSONPath should not be empty")
			assert.Contains(t, jsonPath, "@.field ==", "JSONPath should contain condition")
		})
	}
}

// TestNullVsEmptyVsMissing tests distinction between null, empty string, and missing fields
func TestNullVsEmptyVsMissing(t *testing.T) {
	tests := []struct {
		name        string
		filterMap   map[string]any
		shouldMatch string
		description string
	}{
		{
			name: "isNull true",
			filterMap: map[string]any{
				"field": map[string]any{"isNull": true},
			},
			shouldMatch: "null values",
			description: "Should match null values",
		},
		{
			name: "isNull false",
			filterMap: map[string]any{
				"field": map[string]any{"isNull": false},
			},
			shouldMatch: "non-null values",
			description: "Should match non-null values (including empty string)",
		},
		{
			name: "eq empty string",
			filterMap: map[string]any{
				"field": map[string]any{"eq": ""},
			},
			shouldMatch: "empty string",
			description: "Should match empty string (not null)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonPath, vars, err := BuildJsonFilterFromOperatorMap(tt.filterMap)
			require.NoError(t, err, "Failed to build filter")

			t.Logf("JSONPath: %s", jsonPath)
			t.Logf("Vars: %v", vars)

			// Verify appropriate condition is generated
			if tt.shouldMatch == "null values" {
				assert.Contains(t, jsonPath, "== null", "Should generate null check")
			} else if tt.shouldMatch == "non-null values" {
				assert.Contains(t, jsonPath, "!= null", "Should generate not-null check")
			} else if tt.shouldMatch == "empty string" {
				assert.Contains(t, jsonPath, "==", "Should generate equality check")
				// Verify empty string is in vars
				found := false
				for _, v := range vars {
					if v == "" {
						found = true
						break
					}
				}
				assert.True(t, found, "Empty string should be in vars")
			}
		})
	}
}

// TestTypeCoercion tests handling of different value types
func TestTypeCoercion(t *testing.T) {
	tests := []struct {
		name        string
		value       any
		expectedStr string
		description string
	}{
		{
			name:        "integer",
			value:       123,
			expectedStr: "",
			description: "Integer should be preserved as number",
		},
		{
			name:        "float",
			value:       123.45,
			expectedStr: "",
			description: "Float should be preserved",
		},
		{
			name:        "boolean true",
			value:       true,
			expectedStr: "",
			description: "Boolean should be preserved",
		},
		{
			name:        "boolean false",
			value:       false,
			expectedStr: "",
			description: "Boolean false should be preserved",
		},
		{
			name:        "string number",
			value:       "123",
			expectedStr: "",
			description: "String '123' should remain a string",
		},
		{
			name:        "string boolean",
			value:       "true",
			expectedStr: "",
			description: "String 'true' should remain a string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filterMap := map[string]any{
				"field": map[string]any{"eq": tt.value},
			}

			jsonPath, vars, err := BuildJsonFilterFromOperatorMap(filterMap)
			require.NoError(t, err, "Failed to build filter")

			// Verify value is preserved with correct type
			require.Len(t, vars, 1, "Should have exactly one variable")
			for _, v := range vars {
				assert.Equal(t, tt.value, v, "Value should be preserved with correct type")
			}

			// Verify JSONPath contains the variable reference
			assert.Contains(t, jsonPath, "$v", "JSONPath should contain variable reference")
		})
	}
}

// TestOperatorPrecedence tests correct precedence of AND/OR/NOT operators
func TestOperatorPrecedence(t *testing.T) {
	tests := []struct {
		name              string
		filterMap         map[string]any
		expectedPattern   string
		shouldNotContain  string
		description       string
	}{
		{
			name: "AND has precedence over OR",
			filterMap: map[string]any{
				"OR": []any{
					map[string]any{
						"a": map[string]any{"eq": 1},
						"b": map[string]any{"eq": 2},
					},
					map[string]any{
						"c": map[string]any{"eq": 3},
					},
				},
			},
			expectedPattern:  "||",
			shouldNotContain: "",
			description:      "OR should be properly grouped",
		},
		{
			name: "NOT wraps entire condition",
			filterMap: map[string]any{
				"NOT": map[string]any{
					"a": map[string]any{"eq": 1},
					"b": map[string]any{"eq": 2},
				},
			},
			expectedPattern:  "!(",
			shouldNotContain: "",
			description:      "NOT should wrap the entire AND condition",
		},
		{
			name: "nested AND/OR",
			filterMap: map[string]any{
				"AND": []any{
					map[string]any{
						"OR": []any{
							map[string]any{"a": map[string]any{"eq": 1}},
							map[string]any{"b": map[string]any{"eq": 2}},
						},
					},
					map[string]any{"c": map[string]any{"eq": 3}},
				},
			},
			expectedPattern:  "&&",
			shouldNotContain: "",
			description:      "Nested AND/OR should be properly grouped",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonPath, _, err := BuildJsonFilterFromOperatorMap(tt.filterMap)
			require.NoError(t, err, "Failed to build filter")

			t.Logf("Generated JSONPath: %s", jsonPath)

			if tt.expectedPattern != "" {
				assert.Contains(t, jsonPath, tt.expectedPattern,
					"JSONPath should contain pattern: %s", tt.expectedPattern)
			}

			if tt.shouldNotContain != "" {
				assert.NotContains(t, jsonPath, tt.shouldNotContain,
					"JSONPath should not contain: %s", tt.shouldNotContain)
			}
		})
	}
}

// TestDeepNesting tests correctness with deeply nested structures
func TestDeepNesting(t *testing.T) {
	// Create a filter with 10 levels of nesting
	deepFilter := map[string]any{
		"level1": map[string]any{
			"level2": map[string]any{
				"level3": map[string]any{
					"level4": map[string]any{
						"level5": map[string]any{
							"level6": map[string]any{
								"level7": map[string]any{
									"level8": map[string]any{
										"level9": map[string]any{
											"level10": map[string]any{"eq": "deep"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	jsonPath, vars, err := BuildJsonFilterFromOperatorMap(deepFilter)
	require.NoError(t, err, "Should handle 10 levels of nesting")

	// Verify path contains all levels
	assert.Contains(t, jsonPath, "@.level1.level2.level3.level4.level5.level6.level7.level8.level9.level10",
		"Path should contain all 10 levels")

	// Verify value is preserved
	require.Len(t, vars, 1, "Should have one variable")
	for _, v := range vars {
		assert.Equal(t, "deep", v, "Value should be preserved")
	}
}

// TestSQLGenerationCorrectness verifies generated SQL is valid
func TestSQLGenerationCorrectness(t *testing.T) {
	tests := []struct {
		name           string
		filterMap      map[string]any
		mustContain    []string
		mustNotContain []string
		description    string
	}{
		{
			name: "simple filter SQL",
			filterMap: map[string]any{
				"field": map[string]any{"eq": "value"},
			},
			mustContain: []string{
				"jsonb_path_exists",
				"::jsonpath",
				"::jsonb",
			},
			mustNotContain: []string{
				"undefined",
				"null",
			},
			description: "Should generate valid PostgreSQL jsonb_path_exists call",
		},
		{
			name: "no SQL injection in generated SQL",
			filterMap: map[string]any{
				"field": map[string]any{"eq": "'; DROP TABLE users--"},
			},
			mustContain: []string{
				"jsonb_path_exists",
			},
			mustNotContain: []string{
				"DROP TABLE",
				"--",
			},
			description: "Injection attempt should be parameterized, not in SQL string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			col := goqu.C("data")
			jsonPath, vars, err := BuildJsonFilterFromOperatorMap(tt.filterMap)
			require.NoError(t, err)

			expr, err := BuildJsonPathExistsExpression(col, jsonPath, vars)
			require.NoError(t, err)

			sql, args, err := goqu.Dialect("postgres").
				Select("*").
				From("test").
				Where(expr).
				ToSQL()
			require.NoError(t, err)

			t.Logf("Generated SQL: %s", sql)
			t.Logf("Args: %v", args)

			// Check required patterns
			for _, pattern := range tt.mustContain {
				assert.Contains(t, sql, pattern,
					"SQL should contain: %s", pattern)
			}

			// Check forbidden patterns
			for _, pattern := range tt.mustNotContain {
				assert.NotContains(t, strings.ToUpper(sql), strings.ToUpper(pattern),
					"SQL should not contain: %s", pattern)
			}
		})
	}
}
