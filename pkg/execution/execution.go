package execution

import (
	"context"
	"fmt"
)

type Dialect string

// Executor execute based on graphql.FieldContext & graphql.OperationContext
// The executor holds all drivers that support execution
type Executor struct {
	drivers map[string]Driver
}

func NewExecutor(drivers map[string]Driver) Executor {
	return Executor{
		drivers: drivers,
	}
}

func (e Executor) Get(dialect string) (Driver, error) {
	d, ok := e.drivers[dialect]
	if !ok {
		return nil, fmt.Errorf("missing dialect driver: %s", dialect)
	}
	return d, nil
}

func (e Executor) Scan(ctx context.Context, dialect string, model interface{}) error {
	d, err := e.Get(dialect)
	if err != nil {
		return err
	}
	return d.Scan(ctx, model)
}
