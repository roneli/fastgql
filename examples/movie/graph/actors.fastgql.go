package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"

	"github.com/roneli/fastgql/examples/movie/graph/generated"
	"github.com/roneli/fastgql/examples/movie/graph/model"
)

func (r *movieResolver) Actors(ctx context.Context, obj *model.Movie, limit *int, offset *int, orderBy []*model.ActorOrdering, filter *model.ActorFilterInput) ([]*model.Actor, error) {
	var data []*model.Actor
	if err := r.Executor.Scan(ctx, "postgres", &data); err != nil {
		return nil, err
	}
	return data, nil
}
func (r *movieResolver) ActorsAggregate(ctx context.Context, obj *model.Movie) (*model.ActorsAggregate, error) {
	var data *model.ActorsAggregate
	if err := r.Executor.Scan(ctx, "postgres", &data); err != nil {
		return nil, err
	}
	return data, nil
}

// Movie returns generated.MovieResolver implementation.
func (r *Resolver) Movie() generated.MovieResolver { return &movieResolver{r} }

type movieResolver struct{ *Resolver }
