package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"github.com/roneli/fastgql/builders"
	"github.com/roneli/fastgql/builders/sql"
	"github.com/roneli/fastgql/example/graph/generated"
	"github.com/roneli/fastgql/example/graph/model"

	"github.com/99designs/gqlgen/graphql"
	"github.com/georgysavva/scany/pgxscan"
)

func (r *queryResolver) Posts(ctx context.Context, limit *int, offset *int, orderBy *model.PostOrdering, filter *model.PostFilterInput) ([]*model.Post, error) {
	opCtx := graphql.GetOperationContext(ctx)
	fCtx := graphql.GetFieldContext(ctx)

	builder, _ := sql.NewBuilder(r.Cfg, fCtx.Field.Field)
	err := builders.BuildQuery(&builder, fCtx.Field.Field, opCtx.Variables)
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

	var data []*model.Post
	if err := pgxscan.ScanAll(&data, rows); err != nil {
		return nil, err
	}
	return data, nil

}

func (r *queryResolver) Users(ctx context.Context, limit *int, offset *int, orderBy *model.UserOrdering, filter *model.UserFilterInput) ([]*model.User, error) {
	opCtx := graphql.GetOperationContext(ctx)
	fCtx := graphql.GetFieldContext(ctx)

	builder, _ := sql.NewBuilder(r.Cfg, fCtx.Field.Field)
	err := builders.BuildQuery(&builder, fCtx.Field.Field, opCtx.Variables)
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

	var data []*model.User
	if err := pgxscan.ScanAll(&data, rows); err != nil {
		return nil, err
	}
	return data, nil

}

func (r *queryResolver) Categories(ctx context.Context, limit *int, offset *int, orderBy *model.CategoryOrdering, filter *model.CategoryFilterInput) ([]*model.Category, error) {
	opCtx := graphql.GetOperationContext(ctx)
	fCtx := graphql.GetFieldContext(ctx)

	builder, _ := sql.NewBuilder(r.Cfg, fCtx.Field.Field)
	err := builders.BuildQuery(&builder, fCtx.Field.Field, opCtx.Variables)
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

	var data []*model.Category
	if err := pgxscan.ScanAll(&data, rows); err != nil {
		return nil, err
	}
	return data, nil

}

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type queryResolver struct{ *Resolver }
