package sql

import (
	"testing"

	"github.com/doug-martin/goqu/v9/exp"
	"github.com/stretchr/testify/require"
)

func TestJSONPathBuilder_BuildCondition(t *testing.T) {
	tests := []struct {
		name      string
		condition *ConditionExpr
		wantCond  string
		wantVars  map[string]any
	}{
		{
			name: "eq_condition",
			condition: &ConditionExpr{
				Path:  "color",
				Op:    JSONPathEq,
				Value: "red",
			},
			wantCond: "@.color == $v0",
			wantVars: map[string]any{"v0": "red"},
		},
		{
			name: "neq_condition",
			condition: &ConditionExpr{
				Path:  "status",
				Op:    JSONPathNeq,
				Value: "inactive",
			},
			wantCond: "@.status != $v0",
			wantVars: map[string]any{"v0": "inactive"},
		},
		{
			name: "is_null_true",
			condition: &ConditionExpr{
				Path:   "field",
				IsNull: ptrBool(true),
			},
			wantCond: "@.field == null",
			wantVars: map[string]any{},
		},
		{
			name: "is_null_false",
			condition: &ConditionExpr{
				Path:   "field",
				IsNull: ptrBool(false),
			},
			wantCond: "@.field != null",
			wantVars: map[string]any{},
		},
		{
			name: "regex_pattern",
			condition: &ConditionExpr{
				Path:  "name",
				Regex: `"^test"`,
			},
			wantCond: `@.name like_regex "^test"`,
			wantVars: map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewJSONPathBuilder()
			cond := builder.Build(tt.condition)
			require.Equal(t, tt.wantCond, cond)
			require.Equal(t, tt.wantVars, builder.Vars())
		})
	}
}

func TestJSONPathBuilder_BuildLogical(t *testing.T) {
	tests := []struct {
		name     string
		expr     *LogicalExpr
		wantCond string
	}{
		{
			name: "and_two_conditions",
			expr: &LogicalExpr{
				Op: JSONPathAnd,
				Children: []JSONPathExpr{
					&ConditionExpr{Path: "a", Op: JSONPathEq, Value: 1},
					&ConditionExpr{Path: "b", Op: JSONPathGt, Value: 2},
				},
			},
			wantCond: "@.a == $v0 && @.b > $v1",
		},
		{
			name: "or_two_conditions",
			expr: &LogicalExpr{
				Op: JSONPathOr,
				Children: []JSONPathExpr{
					&ConditionExpr{Path: "a", Op: JSONPathEq, Value: 1},
					&ConditionExpr{Path: "b", Op: JSONPathEq, Value: 2},
				},
			},
			wantCond: "(@.a == $v0 || @.b == $v1)",
		},
		{
			name: "negated_and",
			expr: &LogicalExpr{
				Op: JSONPathAnd,
				Children: []JSONPathExpr{
					&ConditionExpr{Path: "status", Op: JSONPathEq, Value: "active"},
				},
				Negate: true,
			},
			wantCond: "!(@.status == $v0)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewJSONPathBuilder()
			cond := builder.Build(tt.expr)
			require.Equal(t, tt.wantCond, cond)
		})
	}
}

func TestInvertedJSONPathOps(t *testing.T) {
	tests := []struct {
		op       JSONPathOp
		expected JSONPathOp
		exists   bool
	}{
		{JSONPathEq, JSONPathNeq, true},
		{JSONPathNeq, JSONPathEq, true},
		{JSONPathGt, JSONPathLte, true},
		{JSONPathGte, JSONPathLt, true},
		{JSONPathLt, JSONPathGte, true},
		{JSONPathLte, JSONPathGt, true},
		{JSONPathLikeRegex, "", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.op), func(t *testing.T) {
			result, ok := invertedJSONPathOp[tt.op]
			require.Equal(t, tt.exists, ok)
			if tt.exists {
				require.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestConvertFilterMapWithArrayOperators(t *testing.T) {
	dialect := GetSQLDialect("postgres")
	col := exp.NewIdentifierExpression("", "test", "data")

	filterMap := map[string]any{
		"tags": map[string]any{
			"any": map[string]any{
				"eq": "featured",
			},
		},
	}

	expr, err := ConvertFilterMapToExpression(col, filterMap, dialect)
	require.NoError(t, err)
	require.NotNil(t, expr)
}

func TestConvertFilterMapCombinesFieldsAndArrays(t *testing.T) {
	dialect := GetSQLDialect("postgres")
	col := exp.NewIdentifierExpression("", "test", "data")

	complexFilter := map[string]any{
		"color": map[string]any{
			"eq": "red",
		},
		"tags": map[string]any{
			"any": map[string]any{
				"eq": "featured",
			},
		},
	}

	expr, err := ConvertFilterMapToExpression(col, complexFilter, dialect)
	require.NoError(t, err)
	require.NotNil(t, expr)
}

func TestConvertFilterMapAndWithArrayOperators(t *testing.T) {
	dialect := GetSQLDialect("postgres")
	col := exp.NewIdentifierExpression("", "test", "data")

	filterMap := map[string]any{
		"AND": []any{
			map[string]any{
				"color": map[string]any{
					"eq": "red",
				},
			},
			map[string]any{
				"tags": map[string]any{
					"any": map[string]any{
						"eq": "featured",
					},
				},
			},
		},
	}

	expr, err := ConvertFilterMapToExpression(col, filterMap, dialect)
	require.NoError(t, err)
	require.NotNil(t, expr)
}

func TestConvertFilterMapAllOperator(t *testing.T) {
	dialect := GetSQLDialect("postgres")
	col := exp.NewIdentifierExpression("", "test", "data")

	filterMap := map[string]any{
		"items": map[string]any{
			"all": map[string]any{
				"status": map[string]any{
					"eq": "active",
				},
			},
		},
	}

	expr, err := ConvertFilterMapToExpression(col, filterMap, dialect)
	require.NoError(t, err)
	require.NotNil(t, expr)
}

func ptrBool(b bool) *bool {
	return &b
}
