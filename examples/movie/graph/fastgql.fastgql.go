package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"

	"github.com/roneli/fastgql/examples/movie/graph/generated"
	"github.com/roneli/fastgql/examples/movie/graph/model"
)

func (r *mutationResolver) CreateActors(ctx context.Context, inputs []model.CreateActorInput) (*model.ActorsPayload, error) {
	var data *model.ActorsPayload
	if err := r.Executor.Scan(ctx, "postgres", &data); err != nil {
		return nil, err
	}
	return data, nil
}
func (r *mutationResolver) DeleteActors(ctx context.Context, cascade *bool, filter *model.ActorFilterInput) (*model.ActorsPayload, error) {
	var data *model.ActorsPayload
	if err := r.Executor.Scan(ctx, "postgres", &data); err != nil {
		return nil, err
	}
	return data, nil
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

type mutationResolver struct{ *Resolver }
