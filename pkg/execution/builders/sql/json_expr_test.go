package sql_test

import (
	"testing"

	"github.com/doug-martin/goqu/v9/exp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/roneli/fastgql/pkg/execution/builders/sql"
)

func TestJSONFilterBuilder_SimpleConditions(t *testing.T) {
	dialect := sql.GetSQLDialect("postgres")
	col := exp.NewIdentifierExpression("", "test", "attributes")

	tests := []struct {
		name    string
		build   func(*sql.JSONFilterBuilder) *sql.JSONFilterBuilder
		wantErr bool
	}{
		{
			name: "single eq condition",
			build: func(b *sql.JSONFilterBuilder) *sql.JSONFilterBuilder {
				expr, _ := sql.JsonExpr("color", "eq", "red")
				return b.Where(expr)
			},
			wantErr: false,
		},
		{
			name: "single neq condition",
			build: func(b *sql.JSONFilterBuilder) *sql.JSONFilterBuilder {
				expr, _ := sql.JsonExpr("color", "neq", "blue")
				return b.Where(expr)
			},
			wantErr: false,
		},
		{
			name: "comparison operators",
			build: func(b *sql.JSONFilterBuilder) *sql.JSONFilterBuilder {
				gt, _ := sql.JsonExpr("size", "gt", 10)
				gte, _ := sql.JsonExpr("count", "gte", 5)
				lt, _ := sql.JsonExpr("price", "lt", 100)
				lte, _ := sql.JsonExpr("weight", "lte", 50)
				return b.Where(gt, gte, lt, lte)
			},
			wantErr: false,
		},
		{
			name: "string operators",
			build: func(b *sql.JSONFilterBuilder) *sql.JSONFilterBuilder {
				prefix, _ := sql.JsonExpr("name", "prefix", "test")
				suffix, _ := sql.JsonExpr("email", "suffix", ".com")
				contains, _ := sql.JsonExpr("desc", "contains", "hello")
				return b.Where(prefix, suffix, contains)
			},
			wantErr: false,
		},
		{
			name: "null checks",
			build: func(b *sql.JSONFilterBuilder) *sql.JSONFilterBuilder {
				isNull, _ := sql.JsonExpr("field1", "isNull", true)
				isNotNull, _ := sql.JsonExpr("field2", "isNull", false)
				return b.Where(isNull, isNotNull)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := sql.NewJSONFilterBuilder(col, dialect)
			tt.build(builder)

			expr, err := builder.Build()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, expr)
		})
	}
}

func TestJSONFilterBuilder_LogicalOperators(t *testing.T) {
	dialect := sql.GetSQLDialect("postgres")
	col := exp.NewIdentifierExpression("", "test", "attributes")

	t.Run("or conditions", func(t *testing.T) {
		colorRed, _ := sql.JsonExpr("color", "eq", "red")
		colorBlue, _ := sql.JsonExpr("color", "eq", "blue")

		builder := sql.NewJSONFilterBuilder(col, dialect)
		expr, err := builder.
			Where(sql.JsonOr(colorRed, colorBlue)).
			Build()

		require.NoError(t, err)
		assert.NotNil(t, expr)
	})

	t.Run("not condition", func(t *testing.T) {
		statusDeleted, _ := sql.JsonExpr("status", "eq", "deleted")

		builder := sql.NewJSONFilterBuilder(col, dialect)
		expr, err := builder.
			Where(sql.JsonNot(statusDeleted)).
			Build()

		require.NoError(t, err)
		assert.NotNil(t, expr)
	})

	t.Run("complex and/or", func(t *testing.T) {
		typeProduct, _ := sql.JsonExpr("type", "eq", "product")
		priceGt100, _ := sql.JsonExpr("price", "gt", 100)
		typeService, _ := sql.JsonExpr("type", "eq", "service")
		priceGt50, _ := sql.JsonExpr("price", "gt", 50)

		builder := sql.NewJSONFilterBuilder(col, dialect)
		expr, err := builder.
			Where(sql.JsonOr(
				sql.JsonAnd(typeProduct, priceGt100),
				sql.JsonAnd(typeService, priceGt50),
			)).
			Build()

		require.NoError(t, err)
		assert.NotNil(t, expr)
	})
}

func TestJSONFilterBuilder_ArrayOperators(t *testing.T) {
	dialect := sql.GetSQLDialect("postgres")
	col := exp.NewIdentifierExpression("", "test", "attributes")

	t.Run("any array condition", func(t *testing.T) {
		eqFeatured, _ := sql.JsonExpr("", "eq", "featured")

		builder := sql.NewJSONFilterBuilder(col, dialect)
		expr, err := builder.
			Where(sql.JsonAny("tags", eqFeatured)).
			Build()

		require.NoError(t, err)
		assert.NotNil(t, expr)
	})

	t.Run("all array condition", func(t *testing.T) {
		statusActive, _ := sql.JsonExpr("status", "eq", "active")

		builder := sql.NewJSONFilterBuilder(col, dialect)
		expr, err := builder.
			Where(sql.JsonAll("items", statusActive)).
			Build()

		require.NoError(t, err)
		assert.NotNil(t, expr)
	})
}

func TestJSONFilterBuilder_EmptyBuilder(t *testing.T) {
	dialect := sql.GetSQLDialect("postgres")
	col := exp.NewIdentifierExpression("", "test", "attributes")

	builder := sql.NewJSONFilterBuilder(col, dialect)
	_, err := builder.Build()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no conditions")
}
