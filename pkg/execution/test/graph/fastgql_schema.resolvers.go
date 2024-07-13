package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.
// Code generated by github.com/99designs/gqlgen version v0.17.41

import (
	"context"

	"github.com/georgysavva/scany/v2/pgxscan"
	pgx "github.com/jackc/pgx/v5"
	"github.com/roneli/fastgql/pkg/execution/builders/sql"
	"github.com/roneli/fastgql/pkg/execution/test/graph/generated"
	"github.com/roneli/fastgql/pkg/execution/test/graph/model"
)

// CreatePosts is the resolver for the createPosts field.
func (r *mutationResolver) CreatePosts(ctx context.Context, inputs []model.CreatePostInput) (*model.PostsPayload, error) {
	var data model.PostsPayload
	q, args, err := sql.BuildQuery(ctx, sql.NewBuilder(r.Cfg))
	if err != nil {
		return nil, err
	}
	if err := sql.ExecuteQuery(ctx, r.Executor, func(rows pgx.Rows) error {
		return pgxscan.ScanOne(&data, rows)
	}, q, args...); err != nil {
		return nil, err
	}
	return &data, nil
}

// DeletePosts is the resolver for the deletePosts field.
func (r *mutationResolver) DeletePosts(ctx context.Context, cascade *bool, filter *model.PostFilterInput) (*model.PostsPayload, error) {
	var data model.PostsPayload
	q, args, err := sql.BuildQuery(ctx, sql.NewBuilder(r.Cfg))
	if err != nil {
		return nil, err
	}
	if err := sql.ExecuteQuery(ctx, r.Executor, func(rows pgx.Rows) error {
		return pgxscan.ScanOne(&data, rows)
	}, q, args...); err != nil {
		return nil, err
	}
	return &data, nil
}

// UpdatePosts is the resolver for the updatePosts field.
func (r *mutationResolver) UpdatePosts(ctx context.Context, input model.UpdatePostInput, filter *model.PostFilterInput) (*model.PostsPayload, error) {
	var data model.PostsPayload
	q, args, err := sql.BuildQuery(ctx, sql.NewBuilder(r.Cfg))
	if err != nil {
		return nil, err
	}
	if err := sql.ExecuteQuery(ctx, r.Executor, func(rows pgx.Rows) error {
		return pgxscan.ScanOne(&data, rows)
	}, q, args...); err != nil {
		return nil, err
	}
	return &data, nil
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

type mutationResolver struct{ *Resolver }
