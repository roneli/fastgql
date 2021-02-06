package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"

	"github.com/roneli/fastgql/example/graph/example/graph/generated"
	"github.com/roneli/fastgql/example/graph/example/graph/model"
)

func (r *queryResolver) Posts(ctx context.Context, limit *int, offset *int, orderBy *model.PostOrdering, filter *model.PostFilterInput) ([]*model.Post, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) Users(ctx context.Context, limit *int, offset *int, orderBy *model.UserOrdering, filter *model.UserFilterInput) ([]*model.User, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) Categories(ctx context.Context, limit *int, offset *int, orderBy *model.CategoryOrdering, filter *model.CategoryFilterInput) ([]*model.Category, error) {
	panic(fmt.Errorf("not implemented"))
}

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type queryResolver struct{ *Resolver }
