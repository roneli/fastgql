package execution

import (
	"context"
	"github.com/jackc/pgx/v4"
)

// Scanner builds a query based on GraphQL query and scans data into out
type Scanner interface {
	Scan(ctx context.Context, out any) error
}

type Driver interface {
	// Scanner is the main function of all drivers, a scanner will read the GraphQL AST from the context.Context
	Scanner
	// Close gracefully requests the driver to close
	Close() error
	// Dialect returns the dialect name of the driver.
	Dialect() string
}

type RowScanner[T any] interface {
	Scan(ctx context.Context, data pgx.Rows) ([]T, error)
}

type RowScan[T any] interface {
	Scan(ctx context.Context, data pgx.Row) (T, error)
}
