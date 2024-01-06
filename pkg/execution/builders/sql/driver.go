package sql

import (
	"context"
	"fmt"
	"reflect"

	"github.com/jackc/pgx/v5"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/roneli/fastgql/pkg/execution/builders"
)

// Driver is a dialect.Driver implementation for SQL based databases.
type Driver struct {
	builder Builder
	pool    *pgxpool.Pool
	dialect string
}

func BuildQuery(ctx context.Context, builder Builder) (string, []any, error) {
	field := builders.CollectFields(ctx, builder.Schema)
	switch builders.GetOperationType(ctx) {
	case builders.QueryOperation:
		return builder.Query(field)
	case builders.InsertOperation:
		return builder.Create(field)
	case builders.DeleteOperation:
		return builder.Delete(field)
	case builders.UpdateOperation:
		return builder.Update(field)
	}
	return "", nil, fmt.Errorf("invalid operation type %s", builders.GetOperationType(ctx))
}

func ExecuteQuery(ctx context.Context, querier pgxscan.Querier, scanner func(rows pgx.Rows) error, q string, args ...any) error {
	rows, err := querier.Query(ctx, q, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	return scanner(rows)
}

func Collect[T any](ctx context.Context, querier pgxscan.Querier, toFunc pgx.RowToFunc[T], q string, args ...any) ([]T, error) {
	rows, err := querier.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows[T](rows, toFunc)
}

// NewDriver creates a new Driver with the given Conn and dialect.
func NewDriver(dialect string, cfg *builders.Config, pool *pgxpool.Pool) *Driver {
	return &Driver{dialect: dialect, builder: NewBuilder(cfg), pool: pool}
}

func (d Driver) Scan(ctx context.Context, model interface{}) error {
	field := builders.CollectFields(ctx, d.builder.Schema)
	var (
		query string
		args  []interface{}
		err   error
	)
	switch builders.GetOperationType(ctx) {
	case builders.QueryOperation:
		query, args, err = d.builder.Query(field)
	case builders.InsertOperation:
		query, args, err = d.builder.Create(field)
	case builders.DeleteOperation:
		query, args, err = d.builder.Delete(field)
	case builders.UpdateOperation:
		query, args, err = d.builder.Update(field)
	}
	if err != nil {
		return err
	}
	rows, err := d.pool.Query(ctx, query, args...)
	if err != nil {
		return err
	}
	if reflect.TypeOf(model).Elem().Kind() != reflect.Slice {
		if err := pgxscan.ScanOne(model, rows); err != nil {
			return err
		}
		return nil
	}
	if err := pgxscan.ScanAll(model, rows); err != nil {
		return err
	}
	return nil
}

func (d Driver) Close() error {
	d.pool.Close()
	return nil
}

func (d Driver) Dialect() string {
	return d.dialect
}
