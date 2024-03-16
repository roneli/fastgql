package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.
// Code generated by github.com/99designs/gqlgen version v0.17.41

import (
	"context"
	"reflect"

	"github.com/georgysavva/scany/v2/pgxscan"
	pgx "github.com/jackc/pgx/v5"
	"github.com/roneli/fastgql/examples/interface/graph/model"
	"github.com/roneli/fastgql/examples/mutations/graph/generated"
	"github.com/roneli/fastgql/pkg/execution"
	"github.com/roneli/fastgql/pkg/execution/builders/sql"
)

// Posts is the resolver for the posts field.
func (r *queryResolver) Posts(ctx context.Context, limit *int, offset *int, orderBy []*model.PostOrdering, filter *model.PostFilterInput) ([]*model.Post, error) {
	var data []*model.Post
	q, args, err := sql.BuildQuery(ctx, sql.NewBuilder(r.Cfg))
	if err != nil {
		return nil, err
	}
	if err := sql.ExecuteQuery(ctx, r.Executor, func(rows pgx.Rows) error {
		return pgxscan.ScanAll(&data, rows)
	}, q, args...); err != nil {
		return nil, err
	}
	return data, nil
}

// Users is the resolver for the users field.
func (r *queryResolver) Users(ctx context.Context, limit *int, offset *int, orderBy []*model.UserOrdering, filter *model.UserFilterInput) ([]*model.User, error) {
	var data []*model.User
	q, args, err := sql.BuildQuery(ctx, sql.NewBuilder(r.Cfg))
	if err != nil {
		return nil, err
	}
	if err := sql.ExecuteQuery(ctx, r.Executor, func(rows pgx.Rows) error {
		return pgxscan.ScanAll(&data, rows)
	}, q, args...); err != nil {
		return nil, err
	}
	return data, nil
}

// Categories is the resolver for the categories field.
func (r *queryResolver) Categories(ctx context.Context, limit *int, offset *int, orderBy []*model.CategoryOrdering, filter *model.CategoryFilterInput) ([]*model.Category, error) {
	var data []*model.Category
	q, args, err := sql.BuildQuery(ctx, sql.NewBuilder(r.Cfg))
	if err != nil {
		return nil, err
	}
	if err := sql.ExecuteQuery(ctx, r.Executor, func(rows pgx.Rows) error {
		return pgxscan.ScanAll(&data, rows)
	}, q, args...); err != nil {
		return nil, err
	}
	return data, nil
}

// PostsAggregate is the resolver for the _postsAggregate field.
func (r *queryResolver) PostsAggregate(ctx context.Context, groupBy []model.PostGroupBy, filter *model.PostFilterInput) ([]*model.PostsAggregate, error) {
	var data []*model.PostsAggregate
	q, args, err := sql.BuildQuery(ctx, sql.NewBuilder(r.Cfg))
	if err != nil {
		return nil, err
	}
	if err := sql.ExecuteQuery(ctx, r.Executor, func(rows pgx.Rows) error {
		return pgxscan.ScanAll(&data, rows)
	}, q, args...); err != nil {
		return nil, err
	}
	return data, nil
}

// UsersAggregate is the resolver for the _usersAggregate field.
func (r *queryResolver) UsersAggregate(ctx context.Context, groupBy []model.UserGroupBy, filter *model.UserFilterInput) ([]*model.UsersAggregate, error) {
	var data []*model.UsersAggregate
	q, args, err := sql.BuildQuery(ctx, sql.NewBuilder(r.Cfg))
	if err != nil {
		return nil, err
	}
	if err := sql.ExecuteQuery(ctx, r.Executor, func(rows pgx.Rows) error {
		return pgxscan.ScanAll(&data, rows)
	}, q, args...); err != nil {
		return nil, err
	}
	return data, nil
}

// CategoriesAggregate is the resolver for the _categoriesAggregate field.
func (r *queryResolver) CategoriesAggregate(ctx context.Context, groupBy []model.CategoryGroupBy, filter *model.CategoryFilterInput) ([]*model.CategoriesAggregate, error) {
	var data []*model.CategoriesAggregate
	q, args, err := sql.BuildQuery(ctx, sql.NewBuilder(r.Cfg))
	if err != nil {
		return nil, err
	}
	if err := sql.ExecuteQuery(ctx, r.Executor, func(rows pgx.Rows) error {
		return pgxscan.ScanAll(&data, rows)
	}, q, args...); err != nil {
		return nil, err
	}
	return data, nil
}

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type queryResolver struct{ *Resolver }

// !!! WARNING !!!
// The code below was going to be deleted when updating resolvers. It has been copied here so you have
// one last chance to move it out of harms way if you want. There are two reasons this happens:
//   - When renaming or deleting a resolver the old code will be put in here. You can safely delete
//     it when you're done.
//   - You have helper methods in this file. Move them out to keep these resolver files clean.
func (r *queryResolver) Animals(ctx context.Context, limit *int, offset *int, orderBy []*model.AnimalOrdering, filter *model.AnimalFilterInput) ([]model.Animal, error) {
	scanner := execution.NewTypeNameScanner[model.Animal](map[string]reflect.Type{
		"Cat": reflect.TypeOf(model.Cat{}),
		"Dog": reflect.TypeOf(model.Dog{})}, "type")
	q, args, err := sql.BuildQuery(ctx, sql.NewBuilder(r.Cfg))
	if err != nil {
		return nil, err
	}
	return sql.Collect[model.Animal](ctx, r.Executor, func(row pgx.CollectableRow) (model.Animal, error) {
		return scanner.ScanRow(row)
	}, q, args...)
}
func (r *queryResolver) AnimalsAggregate(ctx context.Context, groupBy []model.AnimalGroupBy, filter *model.AnimalFilterInput) ([]*model.AnimalsAggregate, error) {
	var data []*model.AnimalsAggregate
	q, args, err := sql.BuildQuery(ctx, sql.NewBuilder(r.Cfg))
	if err != nil {
		return nil, err
	}
	if err := sql.ExecuteQuery(ctx, r.Executor, func(rows pgx.Rows) error {
		return pgxscan.ScanAll(&data, rows)
	}, q, args...); err != nil {
		return nil, err
	}
	return data, nil
}