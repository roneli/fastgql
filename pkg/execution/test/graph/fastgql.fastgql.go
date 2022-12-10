package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"

	"github.com/roneli/fastgql/pkg/execution/test/graph/generated"
	"github.com/roneli/fastgql/pkg/execution/test/graph/model"
)

func (r *mutationResolver) CreatePosts(ctx context.Context, inputs []model.CreatePostInput) (*model.PostsPayload, error) {
	var data *model.PostsPayload
	if err := r.Executor.Scan(ctx, "postgres", &data); err != nil {
		return nil, err
	}
	return data, nil
}
func (r *mutationResolver) DeletePosts(ctx context.Context, cascade *bool, filter *model.PostFilterInput) (*model.PostsPayload, error) {
	var data *model.PostsPayload
	if err := r.Executor.Scan(ctx, "postgres", &data); err != nil {
		return nil, err
	}
	return data, nil
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

type mutationResolver struct{ *Resolver }
