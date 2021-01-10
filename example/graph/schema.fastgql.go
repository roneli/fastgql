package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fastgql/builders"
	"fastgql/builders/sql"
	"fastgql/example/graph/generated"
	"fastgql/example/graph/model"

	"github.com/99designs/gqlgen/graphql"
	"github.com/georgysavva/scany/pgxscan"
)

func (r *queryResolver) Hero(ctx context.Context, episode *model.Episode, limit *int, offset *int, filter *model.CharacterFilterInput) ([]model.Character, error) {
	opCtx := graphql.GetOperationContext(ctx)
	fCtx := graphql.GetFieldContext(ctx)

	builder := sql.NewBuilder(fCtx.Field.Name)
	err := builders.CollectFields(&builder, fCtx.Field.Field, opCtx.Variables)
	if err != nil {
		return nil, err
	}

	q, args, err := builder.Query()
	if err != nil {
		return nil, err
	}
	rows, err := r.Sql.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	var data []model.Character

	if err := pgxscan.ScanAll(&data, rows); err != nil {
		return nil, err
	}
	return data, nil
}

func (r *queryResolver) Human(ctx context.Context, id string, filter *model.HumanFilterInput) (*model.Human, error) {
	opCtx := graphql.GetOperationContext(ctx)
	fCtx := graphql.GetFieldContext(ctx)

	builder := sql.NewBuilder(fCtx.Field.Name)
	err := builders.CollectFields(&builder, fCtx.Field.Field, opCtx.Variables)
	if err != nil {
		return nil, err
	}

	q, args, err := builder.Query()
	if err != nil {
		return nil, err
	}
	rows, err := r.Sql.Query(ctx, q, args...)
	var data *model.Human

	if err := pgxscan.ScanAll(&data, rows); err != nil {
		return nil, err
	}
	return data, nil
}

func (r *queryResolver) Droid(ctx context.Context, id string, filter *model.DroidFilterInput) (*model.Droid, error) {
	opCtx := graphql.GetOperationContext(ctx)
	fCtx := graphql.GetFieldContext(ctx)

	builder := sql.NewBuilder(fCtx.Field.Name)
	err := builders.CollectFields(&builder, fCtx.Field.Field, opCtx.Variables)
	if err != nil {
		return nil, err
	}

	q, args, err := builder.Query()
	if err != nil {
		return nil, err
	}
	rows, err := r.Sql.Query(ctx, q, args...)
	var data *model.Droid

	if err := pgxscan.ScanAll(&data, rows); err != nil {
		return nil, err
	}
	return data, nil
}

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type queryResolver struct{ *Resolver }
