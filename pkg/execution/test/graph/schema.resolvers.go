package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.
// Code generated by github.com/99designs/gqlgen version v0.17.40-dev

import (
	"context"

	"github.com/georgysavva/scany/v2/pgxscan"
	pgx "github.com/jackc/pgx/v5"
	"github.com/roneli/fastgql/pkg/execution/builders/sql"
	"github.com/roneli/fastgql/pkg/execution/test/graph/generated"
	"github.com/roneli/fastgql/pkg/execution/test/graph/model"
)

// Posts is the resolver for the posts field.
func (r *queryResolver) Posts(ctx context.Context, limit *int, offset *int, orderBy []*model.PostOrdering, filter *model.PostFilterInput) ([]*model.Post, error) {
	var data []*model.Post
	q, args, err := sql.BuildQuery(ctx, sql.NewBuilder(r.Cfg))
	if err != nil {
		return nil, err
	}
	if err := sql.ExecuteQuery(ctx, nil, func(rows pgx.Rows) error {
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
	if err := sql.ExecuteQuery(ctx, nil, func(rows pgx.Rows) error {
		return pgxscan.ScanAll(&data, rows)
	}, q, args...); err != nil {
		return nil, err
	}
	return data, nil
}

// Categories is the resolver for the categories field.
func (r *queryResolver) Categories(ctx context.Context) ([]*model.Category, error) {
	var data []*model.Category
	q, args, err := sql.BuildQuery(ctx, sql.NewBuilder(r.Cfg))
	if err != nil {
		return nil, err
	}
	if err := sql.ExecuteQuery(ctx, nil, func(rows pgx.Rows) error {
		return pgxscan.ScanAll(&data, rows)
	}, q, args...); err != nil {
		return nil, err
	}
	return data, nil
}

// PostsAggregate is the resolver for the _postsAggregate field.
func (r *queryResolver) PostsAggregate(ctx context.Context) (*model.PostsAggregate, error) {
	var data *model.PostsAggregate
	q, args, err := sql.BuildQuery(ctx, sql.NewBuilder(r.Cfg))
	if err != nil {
		return nil, err
	}
	if err := sql.ExecuteQuery(ctx, nil, func(rows pgx.Rows) error {
		return pgxscan.ScanAll(&data, rows)
	}, q, args...); err != nil {
		return nil, err
	}
	return data, nil
}

// UsersAggregate is the resolver for the _usersAggregate field.
func (r *queryResolver) UsersAggregate(ctx context.Context) (*model.UsersAggregate, error) {
	var data *model.UsersAggregate
	q, args, err := sql.BuildQuery(ctx, sql.NewBuilder(r.Cfg))
	if err != nil {
		return nil, err
	}
	if err := sql.ExecuteQuery(ctx, nil, func(rows pgx.Rows) error {
		return pgxscan.ScanAll(&data, rows)
	}, q, args...); err != nil {
		return nil, err
	}
	return data, nil
}

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type queryResolver struct{ *Resolver }
