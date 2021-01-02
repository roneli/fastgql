package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fastgql/example/graph/generated"
	"fastgql/example/graph/model"
	"fmt"
)

func (r *queryResolver) Hero(ctx context.Context, episode *model.Episode, limit *int, offset *int, filter *model.CharacterFilterInput) ([]model.Character, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) Human(ctx context.Context, id string, filter *model.HumanFilterInput) (*model.Human, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) Droid(ctx context.Context, id string, filter *model.DroidFilterInput) (*model.Droid, error) {
	panic(fmt.Errorf("not implemented"))
}

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type queryResolver struct{ *Resolver }
