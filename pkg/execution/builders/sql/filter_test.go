package sql

import (
	"testing"

	"github.com/doug-martin/goqu/v9"
	"github.com/stretchr/testify/assert"
)

func TestOperators(t *testing.T) {
	table := goqu.T("users").As("u")

	tests := []struct {
		name        string
		operator    string
		key         string
		value       interface{}
		wantContain string
	}{
		// Eq operator
		{"eq_string", "eq", "name", "Alice", `"u"."name" = 'Alice'`},
		{"eq_int", "eq", "id", 42, `"u"."id" = 42`},
		{"eq_float", "eq", "score", 3.14, `"u"."score" = 3.14`},

		// Neq operator
		{"neq_string", "neq", "name", "Bob", `"u"."name" != 'Bob'`},
		{"neq_int", "neq", "id", 1, `"u"."id" != 1`},

		// Gt operator
		{"gt_int", "gt", "age", 18, `"u"."age" > 18`},
		{"gt_float", "gt", "score", 3.5, `"u"."score" > 3.5`},

		// Gte operator
		{"gte_int", "gte", "age", 21, `"u"."age" >= 21`},

		// Lt operator
		{"lt_int", "lt", "age", 65, `"u"."age" < 65`},

		// Lte operator
		{"lte_int", "lte", "age", 100, `"u"."age" <= 100`},

		// Like operator
		{"like_pattern", "like", "name", "%Alice%", `"u"."name" LIKE '%Alice%'`},
		{"like_prefix", "like", "name", "A%", `"u"."name" LIKE 'A%'`},
		{"like_suffix", "like", "name", "%a", `"u"."name" LIKE '%a'`},

		// ILike operator
		{"ilike_pattern", "ilike", "name", "%alice%", `"u"."name" ILIKE '%alice%'`},

		// In operator
		{"in_strings", "in", "status", []string{"active", "pending"}, `"u"."status" IN ('active', 'pending')`},
		{"in_ints", "in", "id", []int{1, 2, 3}, `"u"."id" IN (1, 2, 3)`},

		// NotIn operator
		{"notIn_strings", "notIn", "status", []string{"deleted", "banned"}, `"u"."status" NOT IN ('deleted', 'banned')`},

		// Prefix operator
		{"prefix_string", "prefix", "name", "Dr.", `"u"."name" LIKE 'Dr.%'`},

		// Suffix operator
		{"suffix_string", "suffix", "email", "@gmail.com", `"u"."email" LIKE '%@gmail.com'`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op, ok := defaultOperators[tt.operator]
			assert.True(t, ok, "operator %s should exist", tt.operator)

			expr := op(table, tt.key, tt.value)
			sql, _, err := goqu.Dialect("postgres").Select().Where(expr).ToSQL()
			assert.NoError(t, err)
			assert.Contains(t, sql, tt.wantContain)
		})
	}
}

func TestIsNullOperator(t *testing.T) {
	table := goqu.T("users").As("u")

	tests := []struct {
		name        string
		value       interface{}
		wantContain string
	}{
		{"is_null_true", true, `"u"."deleted_at" IS NULL`},
		{"is_null_false", false, `"u"."deleted_at" IS NOT NULL`},
		{"is_null_string_true", "true", `"u"."deleted_at" IS NULL`},
		{"is_null_string_false", "false", `"u"."deleted_at" IS NOT NULL`},
		{"is_null_int_1", 1, `"u"."deleted_at" IS NULL`},
		{"is_null_int_0", 0, `"u"."deleted_at" IS NOT NULL`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op := defaultOperators["isNull"]
			expr := op(table, "deleted_at", tt.value)
			sql, _, err := goqu.Dialect("postgres").Select().Where(expr).ToSQL()
			assert.NoError(t, err)
			assert.Contains(t, sql, tt.wantContain)
		})
	}
}

func TestAllDefaultOperatorsExist(t *testing.T) {
	expectedOperators := []string{
		"eq", "neq", "like", "ilike", "notIn", "in",
		"isNull", "gt", "gte", "lte", "lt", "prefix", "suffix",
	}

	for _, opName := range expectedOperators {
		t.Run(opName, func(t *testing.T) {
			_, ok := defaultOperators[opName]
			assert.True(t, ok, "operator %s should exist in defaultOperators", opName)
		})
	}

	// Verify count matches
	assert.Len(t, defaultOperators, len(expectedOperators))
}

func TestOperatorEdgeCases(t *testing.T) {
	table := goqu.T("test").As("t")

	t.Run("empty_string_eq", func(t *testing.T) {
		expr := defaultOperators["eq"](table, "name", "")
		sql, _, err := goqu.Dialect("postgres").Select().Where(expr).ToSQL()
		assert.NoError(t, err)
		assert.Contains(t, sql, `"t"."name" = ''`)
	})

	t.Run("special_chars_like", func(t *testing.T) {
		expr := defaultOperators["like"](table, "name", "%O'Brien%")
		sql, _, err := goqu.Dialect("postgres").Select().Where(expr).ToSQL()
		assert.NoError(t, err)
		assert.Contains(t, sql, "LIKE")
	})

	t.Run("empty_slice_in", func(t *testing.T) {
		expr := defaultOperators["in"](table, "id", []int{})
		sql, _, err := goqu.Dialect("postgres").Select().Where(expr).ToSQL()
		assert.NoError(t, err)
		// Empty IN should still generate valid SQL
		assert.Contains(t, sql, `"t"."id" IN`)
	})

	t.Run("single_item_in", func(t *testing.T) {
		expr := defaultOperators["in"](table, "id", []int{1})
		sql, _, err := goqu.Dialect("postgres").Select().Where(expr).ToSQL()
		assert.NoError(t, err)
		assert.Contains(t, sql, `"t"."id" IN (1)`)
	})
}
