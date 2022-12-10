package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"

	"github.com/roneli/fastgql/examples/movie/graph/generated"
	"github.com/roneli/fastgql/examples/movie/graph/model"
	"github.com/roneli/fastgql/pkg/execution/builders"
)

func (r *languageResolver) Stuff(ctx context.Context, obj *model.Language, limit *int, offset *int, orderBy []*model.StuffOrdering, filter *model.StuffFilterInput) ([]*model.Stuff, error) {
	var data []*model.Stuff
	ctx = builders.AddRelationFilters(ctx, r.Cfg.Schema, obj)
	if err := r.Executor.Scan(ctx, "mongo", &data); err != nil {
		return nil, err
	}
	return data, nil
}
func (r *queryResolver) Movie(ctx context.Context) (*model.Movie, error) {
	return &model.Movie{}, nil
}
func (r *queryResolver) Actors(ctx context.Context, limit *int, offset *int, orderBy []*model.ActorOrdering, filter *model.ActorFilterInput) ([]*model.Actor, error) {
	var data []*model.Actor
	if err := r.Executor.Scan(ctx, "postgres", &data); err != nil {
		return nil, err
	}
	return data, nil
}
func (r *queryResolver) Films(ctx context.Context, limit *int, offset *int, orderBy []*model.FilmOrdering, filter *model.FilmFilterInput) ([]*model.Film, error) {
	var data []*model.Film
	if err := r.Executor.Scan(ctx, "postgres", &data); err != nil {
		return nil, err
	}
	return data, nil
}
func (r *queryResolver) Language(ctx context.Context, limit *int, offset *int, orderBy []*model.LanguageOrdering, filter *model.LanguageFilterInput) ([]*model.Language, error) {
	var data []*model.Language
	if err := r.Executor.Scan(ctx, "postgres", &data); err != nil {
		return nil, err
	}
	return data, nil
}
func (r *queryResolver) ActorsAggregate(ctx context.Context) (*model.ActorsAggregate, error) {
	var data *model.ActorsAggregate
	if err := r.Executor.Scan(ctx, "postgres", &data); err != nil {
		return nil, err
	}
	return data, nil
}
func (r *queryResolver) FilmsAggregate(ctx context.Context) (*model.FilmsAggregate, error) {
	var data *model.FilmsAggregate
	if err := r.Executor.Scan(ctx, "postgres", &data); err != nil {
		return nil, err
	}
	return data, nil
}
func (r *queryResolver) LanguageAggregate(ctx context.Context) (*model.LanguagesAggregate, error) {
	var data *model.LanguagesAggregate
	if err := r.Executor.Scan(ctx, "postgres", &data); err != nil {
		return nil, err
	}
	return data, nil
}

// Language returns generated.LanguageResolver implementation.
func (r *Resolver) Language() generated.LanguageResolver { return &languageResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type languageResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
