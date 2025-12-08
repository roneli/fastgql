package execution

import (
	"context"
	"fmt"
	"reflect"

	"github.com/roneli/fastgql/pkg/execution/builders"
	"github.com/vektah/gqlparser/v2/ast"
)

// Executor is the main interface for executing GraphQL operations against any database.
// Each database implementation (PostgreSQL, MongoDB, etc.) implements this interface.
type Executor interface {
	// Query executes a read query and scans results into dest
	Query(ctx context.Context, dest any) error
	// QueryWithTypes handles interface types that need type discrimination
	QueryWithTypes(ctx context.Context, dest any, types map[string]reflect.Type, typeKey string) error
	// Mutate executes a create/update/delete mutation and scans results into dest
	Mutate(ctx context.Context, dest any) error
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

// Query routes the query to the appropriate executor based on the type's dialect.
func (m *MultiExecutor) Query(ctx context.Context, dest any) error {
	field := builders.CollectFields(ctx, m.schema)
	dialect := m.getDialectForType(field.TypeDefinition)

	executor, ok := m.executors[dialect]
	if !ok {
		return fmt.Errorf("no executor registered for dialect: %s", dialect)
	}

	return executor.Query(ctx, dest)
}

// QueryWithTypes routes the query to the appropriate executor for interface types.
func (m *MultiExecutor) QueryWithTypes(ctx context.Context, dest any, types map[string]reflect.Type, typeKey string) error {
	field := builders.CollectFields(ctx, m.schema)
	dialect := m.getDialectForType(field.TypeDefinition)

	executor, ok := m.executors[dialect]
	if !ok {
		return fmt.Errorf("no executor registered for dialect: %s", dialect)
	}

	return executor.QueryWithTypes(ctx, dest, types, typeKey)
}

// Mutate routes the mutation to the appropriate executor based on the type's dialect.
func (m *MultiExecutor) Mutate(ctx context.Context, dest any) error {
	field := builders.CollectFields(ctx, m.schema)
	dialect := m.getDialectForType(field.TypeDefinition)

	executor, ok := m.executors[dialect]
	if !ok {
		return fmt.Errorf("no executor registered for dialect: %s", dialect)
	}

	return executor.Mutate(ctx, dest)
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
