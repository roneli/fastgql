package sql

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/roneli/fastgql/pkg/execution/builders"
)

// buildReadQuery builds a read query from the GraphQL context.
func buildReadQuery(ctx context.Context, builder Builder) (string, []any, error) {
	field := builders.CollectFields(ctx, builder.Schema)
	return builder.Query(field)
}

// buildMutationQuery builds a mutation query from the GraphQL context.
func buildMutationQuery(ctx context.Context, builder Builder) (string, []any, error) {
	field := builders.CollectFields(ctx, builder.Schema)
	switch builders.GetOperationType(ctx) {
	case builders.InsertOperation:
		return builder.Create(field)
	case builders.DeleteOperation:
		return builder.Delete(field)
	case builders.UpdateOperation:
		return builder.Update(field)
	}
	return "", nil, fmt.Errorf("invalid mutation operation type %s", builders.GetOperationType(ctx))
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
