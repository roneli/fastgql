package sql

import (
	"testing"

	"github.com/doug-martin/goqu/v9"
	"github.com/stretchr/testify/assert"
)

func TestPostgresDialect_JSONBuildObject(t *testing.T) {
	dialect := PostgresDialect{}

	tests := []struct {
		name        string
		args        []any
		wantContain string
	}{
		{
			name:        "single_pair",
			args:        []any{goqu.L("'name'"), goqu.I("users.name")},
			wantContain: "jsonb_build_object",
		},
		{
			name:        "multiple_pairs",
			args:        []any{goqu.L("'id'"), goqu.I("users.id"), goqu.L("'name'"), goqu.I("users.name")},
			wantContain: "jsonb_build_object",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr := dialect.JSONBuildObject(tt.args...)
			sql, _, err := goqu.Dialect("postgres").Select(expr).ToSQL()
			assert.NoError(t, err)
			assert.Contains(t, sql, tt.wantContain)
		})
	}
}

func TestPostgresDialect_JSONAgg(t *testing.T) {
	dialect := PostgresDialect{}

	expr := dialect.JSONAgg(goqu.I("data"))
	sql, _, err := goqu.Dialect("postgres").Select(expr).ToSQL()
	assert.NoError(t, err)
	assert.Contains(t, sql, "jsonb_agg")
}

func TestPostgresDialect_CoalesceJSON(t *testing.T) {
	dialect := PostgresDialect{}

	tests := []struct {
		name        string
		fallback    string
		wantContain string
	}{
		{
			name:        "empty_array_fallback",
			fallback:    "'[]'::jsonb",
			wantContain: "COALESCE",
		},
		{
			name:        "empty_object_fallback",
			fallback:    "'{}'::jsonb",
			wantContain: "COALESCE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr := dialect.CoalesceJSON(goqu.I("data"), tt.fallback)
			sql, _, err := goqu.Dialect("postgres").Select(expr).ToSQL()
			assert.NoError(t, err)
			assert.Contains(t, sql, tt.wantContain)
		})
	}
}

func TestGetSQLDialect(t *testing.T) {
	tests := []struct {
		name        string
		dialectName string
		wantType    Dialect
	}{
		{
			name:        "postgres",
			dialectName: "postgres",
			wantType:    PostgresDialect{},
		},
		{
			name:        "unknown_defaults_to_postgres",
			dialectName: "unknown",
			wantType:    PostgresDialect{},
		},
		{
			name:        "empty_defaults_to_postgres",
			dialectName: "",
			wantType:    PostgresDialect{},
		},
		{
			name:        "mysql_defaults_to_postgres",
			dialectName: "mysql",
			wantType:    PostgresDialect{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetSQLDialect(tt.dialectName)
			assert.IsType(t, tt.wantType, got)
		})
	}
}

func TestRegisterDialect(t *testing.T) {
	// Create a mock dialect
	type MockDialect struct {
		PostgresDialect
	}

	t.Run("register_new_dialect", func(t *testing.T) {
		RegisterDialect("mock", MockDialect{})
		got := GetSQLDialect("mock")
		assert.IsType(t, MockDialect{}, got)
	})

	t.Run("override_existing_dialect", func(t *testing.T) {
		// Store original
		original := GetSQLDialect("postgres")

		// Override
		RegisterDialect("postgres", MockDialect{})
		got := GetSQLDialect("postgres")
		assert.IsType(t, MockDialect{}, got)

		// Restore
		RegisterDialect("postgres", original)
	})
}

func TestDialectRegistry(t *testing.T) {
	// Verify postgres is in the registry by default
	got := GetSQLDialect("postgres")
	assert.NotNil(t, got)
	assert.IsType(t, PostgresDialect{}, got)
}
