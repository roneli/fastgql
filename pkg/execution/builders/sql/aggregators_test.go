package sql

import (
	"testing"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/roneli/fastgql/pkg/execution/builders"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
)

func TestAggregators(t *testing.T) {
	table := goqu.T("users").As("u")

	fields := []builders.Field{
		{Field: &ast.Field{Name: "id", Alias: "id"}},
		{Field: &ast.Field{Name: "age", Alias: "age"}},
	}

	tests := []struct {
		name        string
		aggregator  string
		expectedSQL string
	}{
		{
			name:        "sum",
			aggregator:  "sum",
			expectedSQL: `SELECT json_build_object('id', SUM("u"."id"), 'age', SUM("u"."age"))`,
		},
		{
			name:        "avg",
			aggregator:  "avg",
			expectedSQL: `SELECT json_build_object('id', AVG("u"."id"), 'age', AVG("u"."age"))`,
		},
		{
			name:        "max",
			aggregator:  "max",
			expectedSQL: `SELECT json_build_object('id', MAX("u"."id"), 'age', MAX("u"."age"))`,
		},
		{
			name:        "min",
			aggregator:  "min",
			expectedSQL: `SELECT json_build_object('id', MIN("u"."id"), 'age', MIN("u"."age"))`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg, ok := defaultAggregatorOperators[tt.aggregator]
			require.True(t, ok, "aggregator %s should exist", tt.aggregator)

			expr, err := agg(table, fields)
			require.NoError(t, err)

			sql, _, err := goqu.Dialect("postgres").Select(expr).ToSQL()
			require.NoError(t, err)
			assert.Equal(t, tt.expectedSQL, sql)
		})
	}
}

func TestAggregatorsSingleField(t *testing.T) {
	table := goqu.T("posts").As("p")
	fields := []builders.Field{
		{Field: &ast.Field{Name: "viewCount", Alias: "viewCount"}},
	}

	tests := []struct {
		name        string
		aggFunc     func(exp.AliasedExpression, []builders.Field) (goqu.Expression, error)
		expectedSQL string
	}{
		{
			name:        "sum_single_field",
			aggFunc:     aggSum,
			expectedSQL: `SELECT json_build_object('viewCount', SUM("p"."view_count"))`,
		},
		{
			name:        "avg_single_field",
			aggFunc:     aggAvg,
			expectedSQL: `SELECT json_build_object('viewCount', AVG("p"."view_count"))`,
		},
		{
			name:        "max_single_field",
			aggFunc:     aggMax,
			expectedSQL: `SELECT json_build_object('viewCount', MAX("p"."view_count"))`,
		},
		{
			name:        "min_single_field",
			aggFunc:     aggMin,
			expectedSQL: `SELECT json_build_object('viewCount', MIN("p"."view_count"))`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := tt.aggFunc(table, fields)
			require.NoError(t, err)

			sql, _, err := goqu.Dialect("postgres").Select(expr).ToSQL()
			require.NoError(t, err)
			assert.Equal(t, tt.expectedSQL, sql)
		})
	}
}

func TestAggregatorsMultipleFields(t *testing.T) {
	table := goqu.T("orders").As("o")
	fields := []builders.Field{
		{Field: &ast.Field{Name: "amount", Alias: "amount"}},
		{Field: &ast.Field{Name: "quantity", Alias: "quantity"}},
		{Field: &ast.Field{Name: "discount", Alias: "discount"}},
	}

	tests := []struct {
		name        string
		aggFunc     func(exp.AliasedExpression, []builders.Field) (goqu.Expression, error)
		expectedSQL string
	}{
		{
			name:        "sum_multiple_fields",
			aggFunc:     aggSum,
			expectedSQL: `SELECT json_build_object('amount', SUM("o"."amount"), 'quantity', SUM("o"."quantity"), 'discount', SUM("o"."discount"))`,
		},
		{
			name:        "avg_multiple_fields",
			aggFunc:     aggAvg,
			expectedSQL: `SELECT json_build_object('amount', AVG("o"."amount"), 'quantity', AVG("o"."quantity"), 'discount', AVG("o"."discount"))`,
		},
		{
			name:        "max_multiple_fields",
			aggFunc:     aggMax,
			expectedSQL: `SELECT json_build_object('amount', MAX("o"."amount"), 'quantity', MAX("o"."quantity"), 'discount', MAX("o"."discount"))`,
		},
		{
			name:        "min_multiple_fields",
			aggFunc:     aggMin,
			expectedSQL: `SELECT json_build_object('amount', MIN("o"."amount"), 'quantity', MIN("o"."quantity"), 'discount', MIN("o"."discount"))`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := tt.aggFunc(table, fields)
			require.NoError(t, err)

			sql, _, err := goqu.Dialect("postgres").Select(expr).ToSQL()
			require.NoError(t, err)
			assert.Equal(t, tt.expectedSQL, sql)
		})
	}
}

func TestAggregatorsEmptyFields(t *testing.T) {
	table := goqu.T("test").As("t")
	fields := []builders.Field{}

	tests := []struct {
		name        string
		aggFunc     func(exp.AliasedExpression, []builders.Field) (goqu.Expression, error)
		expectedSQL string
	}{
		{
			name:        "sum_empty_fields",
			aggFunc:     aggSum,
			expectedSQL: `SELECT json_build_object()`,
		},
		{
			name:        "avg_empty_fields",
			aggFunc:     aggAvg,
			expectedSQL: `SELECT json_build_object()`,
		},
		{
			name:        "max_empty_fields",
			aggFunc:     aggMax,
			expectedSQL: `SELECT json_build_object()`,
		},
		{
			name:        "min_empty_fields",
			aggFunc:     aggMin,
			expectedSQL: `SELECT json_build_object()`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := tt.aggFunc(table, fields)
			require.NoError(t, err)

			sql, _, err := goqu.Dialect("postgres").Select(expr).ToSQL()
			require.NoError(t, err)
			assert.Equal(t, tt.expectedSQL, sql)
		})
	}
}

func TestAllDefaultAggregatorsExist(t *testing.T) {
	expectedAggregators := []string{"sum", "avg", "max", "min"}

	for _, aggName := range expectedAggregators {
		t.Run(aggName, func(t *testing.T) {
			_, ok := defaultAggregatorOperators[aggName]
			assert.True(t, ok, "aggregator %s should exist in defaultAggregatorOperators", aggName)
		})
	}

	assert.Len(t, defaultAggregatorOperators, len(expectedAggregators))
}

func TestAggregatorSnakeCaseConversion(t *testing.T) {
	table := goqu.T("test").As("t")
	fields := []builders.Field{
		{Field: &ast.Field{Name: "createdAt", Alias: "createdAt"}},
		{Field: &ast.Field{Name: "updatedAt", Alias: "updatedAt"}},
		{Field: &ast.Field{Name: "userId", Alias: "userId"}},
	}

	// Verify snake_case conversion for column names in SQL
	expectedSQL := `SELECT json_build_object('createdAt', SUM("t"."created_at"), 'updatedAt', SUM("t"."updated_at"), 'userId', SUM("t"."user_id"))`

	expr, err := aggSum(table, fields)
	require.NoError(t, err)

	sql, _, err := goqu.Dialect("postgres").Select(expr).ToSQL()
	require.NoError(t, err)
	assert.Equal(t, expectedSQL, sql)
}
