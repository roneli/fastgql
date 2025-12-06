package execution

import (
	"context"
	"fmt"
	"reflect"

	"github.com/roneli/fastgql/pkg/execution/builders"
	"github.com/vektah/gqlparser/v2/ast"
)

// Executor is the main interface for executing GraphQL queries against any database.
// Each database implementation (PostgreSQL, MongoDB, etc.) implements this interface.
type Executor interface {
	// Execute builds and runs a query based on GraphQL context, scanning into dest
	Execute(ctx context.Context, dest any) error
	// ExecuteWithTypes handles interface types that need type discrimination
	ExecuteWithTypes(ctx context.Context, dest any, types map[string]reflect.Type, typeKey string) error
	// Close closes the underlying connection
	Close() error
	// Dialect returns the database dialect name
	Dialect() string
}

// MultiExecutor routes queries to the appropriate executor based on the type's dialect.
// It reads the dialect from the @table directive on each GraphQL type.
type MultiExecutor struct {
	executors      map[string]Executor
	schema         *ast.Schema
	defaultDialect string
}

// NewMultiExecutor creates a new MultiExecutor with the given schema and default dialect.
func NewMultiExecutor(schema *ast.Schema, defaultDialect string) *MultiExecutor {
	return &MultiExecutor{
		executors:      make(map[string]Executor),
		schema:         schema,
		defaultDialect: defaultDialect,
	}
}

// Register adds an executor for a specific dialect.
func (m *MultiExecutor) Register(dialect string, executor Executor) {
	m.executors[dialect] = executor
}

// Execute routes the query to the appropriate executor based on the type's dialect.
func (m *MultiExecutor) Execute(ctx context.Context, dest any) error {
	field := builders.CollectFields(ctx, m.schema)
	dialect := m.getDialectForType(field.TypeDefinition)

	executor, ok := m.executors[dialect]
	if !ok {
		return fmt.Errorf("no executor registered for dialect: %s", dialect)
	}

	return executor.Execute(ctx, dest)
}

// ExecuteWithTypes routes the query to the appropriate executor for interface types.
func (m *MultiExecutor) ExecuteWithTypes(ctx context.Context, dest any, types map[string]reflect.Type, typeKey string) error {
	field := builders.CollectFields(ctx, m.schema)
	dialect := m.getDialectForType(field.TypeDefinition)

	executor, ok := m.executors[dialect]
	if !ok {
		return fmt.Errorf("no executor registered for dialect: %s", dialect)
	}

	return executor.ExecuteWithTypes(ctx, dest, types, typeKey)
}

// Close closes all registered executors.
func (m *MultiExecutor) Close() error {
	var lastErr error
	for _, exec := range m.executors {
		if err := exec.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// Dialect returns "multi" to indicate this is a multi-executor.
func (m *MultiExecutor) Dialect() string {
	return "multi"
}

// getDialectForType extracts the dialect from the @table directive on a type.
func (m *MultiExecutor) getDialectForType(typeDef *ast.Definition) string {
	if typeDef == nil {
		return m.defaultDialect
	}
	if d := typeDef.Directives.ForName("table"); d != nil {
		if arg := d.Arguments.ForName("dialect"); arg != nil {
			return arg.Value.Raw
		}
	}
	return m.defaultDialect
}
