package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/roneli/fastgql/examples/simple/graph/generated"
	"github.com/roneli/fastgql/examples/simple/graph/model"
	"github.com/roneli/fastgql/pkg/builders"
	"github.com/roneli/fastgql/pkg/builders/sql"
)

func (r *queryResolver) Posts(ctx context.Context, limit *int, offset *int, orderBy []*model.PostOrdering, filter *model.PostFilterInput) ([]*model.Post, error) {
	builder := sql.NewBuilder(r.Cfg)
	q, args, err := builder.Query(builders.CollectFields(ctx))
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
	q, args, err := builder.Query(builders.CollectFields(ctx))
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
func (r *queryResolver) PostsAggregate(ctx context.Context) (*model.PostsAggregate, error) {
	builder := sql.NewBuilder(r.Cfg)
	q, args, err := builder.Aggregate(builders.CollectFields(ctx))
	if err != nil {
		return nil, err
	}
	rows, err := r.Executor.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	var data *model.PostsAggregate
	if err := pgxscan.ScanOne(&data, rows); err != nil {
		return nil, err
	}
	return data, nil
}
func (r *queryResolver) UsersAggregate(ctx context.Context) (*model.UsersAggregate, error) {
	builder := sql.NewBuilder(r.Cfg)
	q, args, err := builder.Aggregate(builders.CollectFields(ctx))
	if err != nil {
		return nil, err
	}
	rows, err := r.Executor.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	var data *model.UsersAggregate
	if err := pgxscan.ScanOne(&data, rows); err != nil {
		return nil, err
	}
	return data, nil
}
func (r *queryResolver) CategoriesAggregate(ctx context.Context) (*model.CategoriesAggregate, error) {
	builder := sql.NewBuilder(r.Cfg)
	q, args, err := builder.Aggregate(builders.CollectFields(ctx))
	if err != nil {
		return nil, err
	}
	rows, err := r.Executor.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	var data *model.CategoriesAggregate
	if err := pgxscan.ScanOne(&data, rows); err != nil {
		return nil, err
	}
	return data, nil
}

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type queryResolver struct{ *Resolver }
