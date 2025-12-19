package sql_test

import (
	"testing"

	"github.com/doug-martin/goqu/v9/exp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/roneli/fastgql/pkg/execution/builders/sql"
)

func TestJSONPathConditionExpr_ToJSONPathString(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		operator  string
		value     any
		wantCond  string
		wantValue any
		wantErr   bool
	}{
		// Standard comparison operators
		{
			name:      "eq operator",
			path:      "color",
			operator:  "eq",
			value:     "red",
			wantCond:  "@.color == $v0",
			wantValue: "red",
			wantErr:   false,
		},
		{
			name:      "neq operator",
			path:      "color",
			operator:  "neq",
			value:     "blue",
			wantCond:  "@.color != $v0",
			wantValue: "blue",
			wantErr:   false,
		},
		{
			name:      "gt operator",
			path:      "size",
			operator:  "gt",
			value:     10,
			wantCond:  "@.size > $v0",
			wantValue: 10,
			wantErr:   false,
		},
		{
			name:      "gte operator",
			path:      "size",
			operator:  "gte",
			value:     10,
			wantCond:  "@.size >= $v0",
			wantValue: 10,
			wantErr:   false,
		},
		{
			name:      "lt operator",
			path:      "size",
			operator:  "lt",
			value:     10,
			wantCond:  "@.size < $v0",
			wantValue: 10,
			wantErr:   false,
		},
		{
			name:      "lte operator",
			path:      "size",
			operator:  "lte",
			value:     10,
			wantCond:  "@.size <= $v0",
			wantValue: 10,
			wantErr:   false,
		},
		{
			name:      "like operator",
			path:      "name",
			operator:  "like",
			value:     "test.*",
			wantCond:  "@.name like_regex $v0",
			wantValue: "test.*",
			wantErr:   false,
		},
		{
			name:      "isNull true",
			path:      "color",
			operator:  "isNull",
			value:     true,
			wantCond:  "@.color == null",
			wantValue: nil,
			wantErr:   false,
		},
		{
			name:      "isNull false",
			path:      "color",
			operator:  "isNull",
			value:     false,
			wantCond:  "@.color != null",
			wantValue: nil,
			wantErr:   false,
		},
		// Regex-based operators
		{
			name:      "prefix operator",
			path:      "color",
			operator:  "prefix",
			value:     "red",
			wantCond:  "@.color like_regex \"^red\"",
			wantValue: nil,
			wantErr:   false,
		},
		{
			name:      "suffix operator",
			path:      "color",
			operator:  "suffix",
			value:     "blue",
			wantCond:  "@.color like_regex \"blue$\"",
			wantValue: nil,
			wantErr:   false,
		},
		{
			name:      "ilike operator",
			path:      "color",
			operator:  "ilike",
			value:     "red",
			wantCond:  "@.color like_regex \"red\" flag \"i\"",
			wantValue: nil,
			wantErr:   false,
		},
		{
			name:      "contains operator",
			path:      "color",
			operator:  "contains",
			value:     "ed",
			wantCond:  "@.color like_regex \"ed\"",
			wantValue: nil,
			wantErr:   false,
		},
		// Regex escaping tests
		{
			name:      "prefix with dot",
			path:      "path",
			operator:  "prefix",
			value:     "test.value",
			wantCond:  "@.path like_regex \"^test\\.value\"",
			wantValue: nil,
			wantErr:   false,
		},
		{
			name:      "prefix with dollar",
			path:      "path",
			operator:  "prefix",
			value:     "test$value",
			wantCond:  "@.path like_regex \"^test\\$value\"",
			wantValue: nil,
			wantErr:   false,
		},
		{
			name:      "ilike with asterisk",
			path:      "name",
			operator:  "ilike",
			value:     "test*value",
			wantCond:  "@.name like_regex \"test\\*value\" flag \"i\"",
			wantValue: nil,
			wantErr:   false,
		},
		{
			name:      "contains with parentheses",
			path:      "name",
			operator:  "contains",
			value:     "test(value)",
			wantCond:  "@.name like_regex \"test\\(value\\)\"",
			wantValue: nil,
			wantErr:   false,
		},
		{
			name:      "suffix with brackets",
			path:      "name",
			operator:  "suffix",
			value:     "test[0]",
			wantCond:  "@.name like_regex \"test\\[0\\]$\"",
			wantValue: nil,
			wantErr:   false,
		},
		{
			name:      "prefix with multiple special chars",
			path:      "path",
			operator:  "prefix",
			value:     "test.value$test*test",
			wantCond:  "@.path like_regex \"^test\\.value\\$test\\*test\"",
			wantValue: nil,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cond, err := sql.NewJSONPathCondition(tt.path, tt.operator, tt.value)
			require.NoError(t, err)

			gotCond, gotValue, err := cond.ToJSONPathString()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantCond, gotCond)
			assert.Equal(t, tt.wantValue, gotValue)
		})
	}
}

func TestJSONPathFilterExpr_Expression(t *testing.T) {
	dialect := sql.GetSQLDialect("postgres")
	col := exp.NewIdentifierExpression("", "test", "attributes")

	tests := []struct {
		name       string
		conditions []struct {
			path     string
			operator string
			value    any
		}
		wantErr bool
	}{
		{
			name: "single eq condition",
			conditions: []struct {
				path     string
				operator string
				value    any
			}{
				{"color", "eq", "red"},
			},
			wantErr: false,
		},
		{
			name: "single prefix condition",
			conditions: []struct {
				path     string
				operator string
				value    any
			}{
				{"color", "prefix", "red"},
			},
			wantErr: false,
		},
		{
			name: "multiple standard operators",
			conditions: []struct {
				path     string
				operator string
				value    any
			}{
				{"color", "eq", "red"},
				{"size", "gt", 10},
			},
			wantErr: false,
		},
		{
			name: "multiple regex operators",
			conditions: []struct {
				path     string
				operator string
				value    any
			}{
				{"color", "prefix", "red"},
				{"name", "contains", "test"},
			},
			wantErr: false,
		},
		{
			name: "mixed standard and regex operators",
			conditions: []struct {
				path     string
				operator string
				value    any
			}{
				{"color", "prefix", "red"},
				{"size", "gt", 10},
				{"name", "ilike", "TEST"},
			},
			wantErr: false,
		},
		{
			name: "all comparison operators",
			conditions: []struct {
				path     string
				operator string
				value    any
			}{
				{"a", "eq", "x"},
				{"b", "neq", "y"},
				{"c", "gt", 1},
				{"d", "gte", 2},
				{"e", "lt", 3},
				{"f", "lte", 4},
			},
			wantErr: false,
		},
		{
			name: "all regex operators",
			conditions: []struct {
				path     string
				operator string
				value    any
			}{
				{"a", "like", ".*"},
				{"b", "prefix", "start"},
				{"c", "suffix", "end"},
				{"d", "ilike", "CASE"},
				{"e", "contains", "middle"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := sql.NewJSONPathFilter(col, dialect)
			for _, cond := range tt.conditions {
				c, err := sql.NewJSONPathCondition(cond.path, cond.operator, cond.value)
				require.NoError(t, err)
				filter.AddCondition(c)
			}

			expr, err := filter.Expression()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, expr)
		})
	}
}
