package execution

import (
	"context"
)

// Scanner builds a query based on GraphQL query and scans data into v
type Scanner interface {
	Scan(ctx context.Context, v any) error
}

type Driver interface {
	// Scanner is the main function of all drivers, a scanner will read the GraphQL AST from the context.Context
	Scanner
	// Close gracefully requests the driver to close
	Close() error
	// Dialect returns the dialect name of the driver.
	Dialect() string
}
