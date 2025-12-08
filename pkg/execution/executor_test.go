package execution

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
)

func TestNewMultiExecutor(t *testing.T) {
	tests := []struct {
		name           string
		defaultDialect string
	}{
		{"postgres_default", "postgres"},
		{"mysql_default", "mysql"},
		{"empty_default", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := &ast.Schema{}
			multi := NewMultiExecutor(schema, tt.defaultDialect)

			require.NotNil(t, multi)
			assert.Equal(t, tt.defaultDialect, multi.defaultDialect)
			assert.NotNil(t, multi.executors)
			assert.Equal(t, schema, multi.schema)
		})
	}
}

func TestMultiExecutor_Register(t *testing.T) {
	schema := &ast.Schema{}
	multi := NewMultiExecutor(schema, "postgres")

	t.Run("register_single", func(t *testing.T) {
		mock := &mockExecutor{dialect: "postgres"}
		multi.Register("postgres", mock)

		assert.Len(t, multi.executors, 1)
	})

	t.Run("register_multiple", func(t *testing.T) {
		multi.Register("mysql", &mockExecutor{dialect: "mysql"})
		multi.Register("snowflake", &mockExecutor{dialect: "snowflake"})

		assert.Len(t, multi.executors, 3)
	})

	t.Run("override_existing", func(t *testing.T) {
		newMock := &mockExecutor{dialect: "postgres-new"}
		multi.Register("postgres", newMock)

		// Should still have 3 executors, just with the postgres one replaced
		assert.Len(t, multi.executors, 3)
	})
}

func TestMultiExecutor_getDialectForType(t *testing.T) {
	schema := &ast.Schema{
		Types: map[string]*ast.Definition{
			"User": {
				Name: "User",
				Directives: ast.DirectiveList{
					{
						Name: "table",
						Arguments: ast.ArgumentList{
							{
								Name:  "name",
								Value: &ast.Value{Raw: "users"},
							},
						},
					},
				},
			},
			"AnalyticsEvent": {
				Name: "AnalyticsEvent",
				Directives: ast.DirectiveList{
					{
						Name: "table",
						Arguments: ast.ArgumentList{
							{
								Name:  "name",
								Value: &ast.Value{Raw: "events"},
							},
							{
								Name:  "dialect",
								Value: &ast.Value{Raw: "snowflake"},
							},
						},
					},
				},
			},
		},
	}

	multi := NewMultiExecutor(schema, "postgres")

	tests := []struct {
		name        string
		typeDef     *ast.Definition
		wantDialect string
	}{
		{
			name:        "nil_type_uses_default",
			typeDef:     nil,
			wantDialect: "postgres",
		},
		{
			name:        "type_without_dialect_uses_default",
			typeDef:     schema.Types["User"],
			wantDialect: "postgres",
		},
		{
			name:        "type_with_dialect",
			typeDef:     schema.Types["AnalyticsEvent"],
			wantDialect: "snowflake",
		},
		{
			name: "type_without_table_directive",
			typeDef: &ast.Definition{
				Name:       "NoTable",
				Directives: ast.DirectiveList{},
			},
			wantDialect: "postgres",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := multi.getDialectForType(tt.typeDef)
			assert.Equal(t, tt.wantDialect, got)
		})
	}
}

// mockExecutor implements Executor interface for testing
type mockExecutor struct {
	dialect string
}

func (m *mockExecutor) Query(_ context.Context, _ any) error {
	return nil
}

func (m *mockExecutor) QueryWithTypes(_ context.Context, _ any, _ map[string]reflect.Type, _ string) error {
	return nil
}

func (m *mockExecutor) Mutate(_ context.Context, _ any) error {
	return nil
}
