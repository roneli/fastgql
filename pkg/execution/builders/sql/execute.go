package sql

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/roneli/fastgql/pkg/execution/builders"
)

// buildQuery builds a query from the GraphQL context using the provided builder.
// This is used by generated resolver code.
func buildQuery(ctx context.Context, builder Builder) (string, []any, error) {
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

// collect executes a query and collects rows using the provided row function.
// This is used by generated resolver code for interface types.
func collect[T any](ctx context.Context, querier pgxscan.Querier, toFunc pgx.RowToFunc[T], q string, args ...any) ([]T, error) {
	rows, err := querier.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows[T](rows, toFunc)
}
