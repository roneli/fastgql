package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/roneli/fastgql/examples/movie/graph/generated"
	"github.com/roneli/fastgql/examples/movie/graph/model"
	"github.com/roneli/fastgql/pkg/builders"
	"github.com/roneli/fastgql/pkg/builders/sql"
)

func (r *movieResolver) Actors(ctx context.Context, obj *model.Movie, limit *int, offset *int, orderBy []*model.ActorOrdering, filter *model.ActorFilterInput) ([]*model.Actor, error) {

	builder := sql.NewBuilder(r.Cfg)
	q, args, err := builder.Query(builders.CollectFields(ctx))
	if err != nil {
		return nil, err
	}
	rows, err := r.Executor.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	var data []*model.Actor
	if err := pgxscan.ScanAll(&data, rows); err != nil {
		return nil, err
	}
	return data, nil

}
func (r *movieResolver) ActorsAggregate(ctx context.Context, obj *model.Movie) (*model.ActorsAggregate, error) {

	builder := sql.NewBuilder(r.Cfg)
	q, args, err := builder.Aggregate(builders.CollectFields(ctx))
	if err != nil {
		return nil, err
	}
	rows, err := r.Executor.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	var data *model.ActorsAggregate
	if err := pgxscan.ScanOne(&data, rows); err != nil {
		return nil, err
	}
	return data, nil

}

// Movie returns generated.MovieResolver implementation.
func (r *Resolver) Movie() generated.MovieResolver { return &movieResolver{r} }

type movieResolver struct{ *Resolver }
