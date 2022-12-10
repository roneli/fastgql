package sql

import (
	"context"
	"reflect"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/roneli/fastgql/pkg/execution/builders"
)

// Driver is a dialect.Driver implementation for SQL based databases.
type Driver struct {
	builder Builder
	pool    *pgxpool.Pool
	dialect string
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
