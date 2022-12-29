package execution

import (
	"context"
)

// Scanner builds a query based on GraphQL query and scans data into v
type Scanner interface {
	Scan(ctx context.Context, v any) error
}

type Driver interface {
	Scanner
	Close() error
	// Dialect returns the dialect name of the driver.
	Dialect() string
}
