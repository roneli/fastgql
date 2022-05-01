package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/roneli/fastgql/examples/simple/graph/generated"
	"github.com/roneli/fastgql/examples/simple/graph/model"
	"github.com/roneli/fastgql/pkg/builders"
	"github.com/roneli/fastgql/pkg/builders/sql"
)

func (r *mutationResolver) CreatePosts(ctx context.Context, inputs []model.CreatePostInput) (*model.PostsPayload, error) {
	builder := sql.NewBuilder(r.Cfg)
	q, args, err := builder.Create(builders.CollectFields(ctx))
	if err != nil {
		return nil, err
	}
	rows, err := r.Executor.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	var data model.PostsPayload
	if err := pgxscan.ScanOne(&data, rows); err != nil {
		return nil, err
	}
	return &data, nil
}
func (r *mutationResolver) DeletePosts(ctx context.Context, cascade *bool, filter *model.PostFilterInput) (*model.PostsPayload, error) {
	builder := sql.NewBuilder(r.Cfg)
	q, args, err := builder.Delete(builders.CollectFields(ctx))
	if err != nil {
		return nil, err
	}
	rows, err := r.Executor.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	var data model.PostsPayload
	if err := pgxscan.ScanOne(&data, rows); err != nil {
		return nil, err
	}
	return &data, nil
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

type mutationResolver struct{ *Resolver }
