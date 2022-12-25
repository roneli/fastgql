package execution

import (
	"context"
)

// Scanner builds a query and scans data into v based on graphql query
type Scanner interface {
	Scan(ctx context.Context, v interface{}) error
}

// Mutator mutates data from a mutation
type Mutator interface {
	Mutate(ctx context.Context, operation string, v interface{}) error
}

type Driver interface {
	Scanner
	Close() error
	// Dialect returns the dialect name of the driver.
	Dialect() string
}
