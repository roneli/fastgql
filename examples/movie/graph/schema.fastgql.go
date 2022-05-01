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

func (r *queryResolver) Movie(ctx context.Context) (*model.Movie, error) {
	return &model.Movie{}, nil
}
func (r *queryResolver) Actors(ctx context.Context, limit *int, offset *int, orderBy []*model.ActorOrdering, filter *model.ActorFilterInput) ([]*model.Actor, error) {
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
func (r *queryResolver) Films(ctx context.Context, limit *int, offset *int, orderBy []*model.FilmOrdering, filter *model.FilmFilterInput) ([]*model.Film, error) {
	builder := sql.NewBuilder(r.Cfg)
	q, args, err := builder.Query(builders.CollectFields(ctx))
	if err != nil {
		return nil, err
	}
	rows, err := r.Executor.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	var data []*model.Film
	if err := pgxscan.ScanAll(&data, rows); err != nil {
		return nil, err
	}
	return data, nil
}
func (r *queryResolver) Language(ctx context.Context, limit *int, offset *int, orderBy []*model.LanguageOrdering, filter *model.LanguageFilterInput) ([]*model.Language, error) {
	builder := sql.NewBuilder(r.Cfg)
	q, args, err := builder.Query(builders.CollectFields(ctx))
	if err != nil {
		return nil, err
	}
	rows, err := r.Executor.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	var data []*model.Language
	if err := pgxscan.ScanAll(&data, rows); err != nil {
		return nil, err
	}
	return data, nil
}
func (r *queryResolver) ActorsAggregate(ctx context.Context) (*model.ActorsAggregate, error) {
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
func (r *queryResolver) FilmsAggregate(ctx context.Context) (*model.FilmsAggregate, error) {
	builder := sql.NewBuilder(r.Cfg)
	q, args, err := builder.Aggregate(builders.CollectFields(ctx))
	if err != nil {
		return nil, err
	}
	rows, err := r.Executor.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	var data *model.FilmsAggregate
	if err := pgxscan.ScanOne(&data, rows); err != nil {
		return nil, err
	}
	return data, nil
}
func (r *queryResolver) LanguageAggregate(ctx context.Context) (*model.LanguagesAggregate, error) {
	builder := sql.NewBuilder(r.Cfg)
	q, args, err := builder.Aggregate(builders.CollectFields(ctx))
	if err != nil {
		return nil, err
	}
	rows, err := r.Executor.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	var data *model.LanguagesAggregate
	if err := pgxscan.ScanOne(&data, rows); err != nil {
		return nil, err
	}
	return data, nil
}

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type queryResolver struct{ *Resolver }
