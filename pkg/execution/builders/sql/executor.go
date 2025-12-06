package sql

import (
	"context"
	"reflect"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/roneli/fastgql/pkg/execution/builders"
)

// Executor implements execution.Executor for PostgreSQL databases.
type Executor struct {
	pool    *pgxpool.Pool
	config  *builders.Config
	builder Builder
	dialect string
}

// NewExecutor creates a new SQL Executor with the given pool and config.
func NewExecutor(pool *pgxpool.Pool, config *builders.Config) *Executor {
	dialect := config.Dialect
	if dialect == "" {
		dialect = "postgres"
	}
	return &Executor{
		pool:    pool,
		config:  config,
		builder: NewBuilder(config),
		dialect: dialect,
	}
}

// Execute builds and runs a query based on GraphQL context, scanning into dest.
func (e *Executor) Execute(ctx context.Context, dest any) error {
	query, args, err := buildQuery(ctx, e.builder)
	if err != nil {
		return err
	}

	rows, err := e.pool.Query(ctx, query, args...)
	if err != nil {
		return err
	}

	// Determine if we're scanning a single row or multiple rows
	destType := reflect.TypeOf(dest)
	if destType.Kind() == reflect.Ptr {
		destType = destType.Elem()
	}

	if destType.Kind() != reflect.Slice {
		return pgxscan.ScanOne(dest, rows)
	}
	return pgxscan.ScanAll(dest, rows)
}

// ExecuteWithTypes handles interface types that need type discrimination.
func (e *Executor) ExecuteWithTypes(ctx context.Context, dest any, types map[string]reflect.Type, typeKey string) error {
	query, args, err := buildQuery(ctx, e.builder)
	if err != nil {
		return err
	}

	scanner := NewTypeNameScanner[any](types, typeKey)
	results, err := collect(ctx, e.pool, func(row pgx.CollectableRow) (any, error) {
		return scanner.ScanRow(row)
	}, query, args...)
	if err != nil {
		return err
	}

	// Set the results into dest using reflection
	destVal := reflect.ValueOf(dest).Elem()
	sliceVal := reflect.MakeSlice(destVal.Type(), len(results), len(results))
	for i, r := range results {
		sliceVal.Index(i).Set(reflect.ValueOf(r))
	}
	destVal.Set(sliceVal)
	return nil
}

// Close closes the connection pool.
func (e *Executor) Close() error {
	e.pool.Close()
	return nil
}

// Dialect returns the SQL dialect name.
func (e *Executor) Dialect() string {
	return e.dialect
}
