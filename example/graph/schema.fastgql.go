package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/roneli/fastgql/builders/sql"
	"github.com/roneli/fastgql/example/graph/generated"
	"github.com/roneli/fastgql/example/graph/model"
)

func (r *queryResolver) Posts(ctx context.Context, limit *int, offset *int, orderBy []*model.PostOrdering, filter *model.PostFilterInput) ([]*model.Post, error) {
	builder := sql.NewBuilder(r.Cfg)
	q, args, err := builder.Query(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.Executor.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	var data []*model.Post
	if err := pgxscan.ScanAll(&data, rows); err != nil {
		return nil, err
	}
	return data, nil
}

func (r *queryResolver) Users(ctx context.Context, limit *int, offset *int, orderBy []*model.UserOrdering, filter *model.UserFilterInput) ([]*model.User, error) {
	builder := sql.NewBuilder(r.Cfg)
	q, args, err := builder.Query(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.Executor.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	var data []*model.User
	if err := pgxscan.ScanAll(&data, rows); err != nil {
		return nil, err
	}
	return data, nil
}

func (r *queryResolver) Categories(ctx context.Context, limit *int, offset *int, orderBy []*model.CategoryOrdering, filter *model.CategoryFilterInput) ([]*model.Category, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) PostsAggregate(ctx context.Context, filter *model.PostFilterInput) (*model.AggregateResult, error) {
	builder := sql.NewBuilder(r.Cfg)
	q, args, err := builder.Aggregate(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.Executor.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	var data *model.AggregateResult
	if err := pgxscan.ScanOne(&data, rows); err != nil {
		return nil, err
	}
	return data, nil
}

func (r *queryResolver) UsersAggregate(ctx context.Context, filter *model.UserFilterInput) (*model.AggregateResult, error) {
	builder := sql.NewBuilder(r.Cfg)
	q, args, err := builder.Aggregate(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.Executor.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	var data *model.AggregateResult
	if err := pgxscan.ScanOne(&data, rows); err != nil {
		return nil, err
	}
	return data, nil
}

func (r *queryResolver) CategoriesAggregate(ctx context.Context, filter *model.CategoryFilterInput) (*model.AggregateResult, error) {
	builder := sql.NewBuilder(r.Cfg)
	q, args, err := builder.Aggregate(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.Executor.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	var data *model.AggregateResult
	if err := pgxscan.ScanOne(&data, rows); err != nil {
		return nil, err
	}
	return data, nil
}

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type queryResolver struct{ *Resolver }
