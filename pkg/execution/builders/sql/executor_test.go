package sql

import (
	"testing"

	"github.com/roneli/fastgql/pkg/execution/builders"
	"github.com/stretchr/testify/assert"
	"github.com/vektah/gqlparser/v2/ast"
)

func TestNewExecutor(t *testing.T) {
	// Note: We can't test with a real pool without a database connection
	// These tests verify the configuration is handled correctly

	t.Run("with_default_dialect", func(t *testing.T) {
		config := &builders.Config{
			Schema: &ast.Schema{},
		}
		executor := NewExecutor(nil, config)
		assert.Equal(t, "postgres", executor.Dialect())
	})

	t.Run("with_custom_dialect", func(t *testing.T) {
		config := &builders.Config{
			Schema:  &ast.Schema{},
			Dialect: "mysql",
		}
		executor := NewExecutor(nil, config)
		assert.Equal(t, "mysql", executor.Dialect())
	})
}

func TestExecutor_Dialect(t *testing.T) {
	tests := []struct {
		name           string
		configDialect  string
		expectedResult string
	}{
		{
			name:           "postgres_default",
			configDialect:  "",
			expectedResult: "postgres",
		},
		{
			name:           "postgres_explicit",
			configDialect:  "postgres",
			expectedResult: "postgres",
		},
		{
			name:           "mysql",
			configDialect:  "mysql",
			expectedResult: "mysql",
		},
		{
			name:           "snowflake",
			configDialect:  "snowflake",
			expectedResult: "snowflake",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &builders.Config{
				Schema:  &ast.Schema{},
				Dialect: tt.configDialect,
			}
			executor := NewExecutor(nil, config)
			assert.Equal(t, tt.expectedResult, executor.Dialect())
		})
	}
}
