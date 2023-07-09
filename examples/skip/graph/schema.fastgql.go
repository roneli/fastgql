package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"

	"github.com/roneli/fastgql/examples/skip/graph/generated"
	"github.com/roneli/fastgql/examples/skip/graph/model"
)

func (r *queryResolver) Person(ctx context.Context) (*model.Person, error) {
	panic(fmt.Errorf("not implemented"))
}

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type queryResolver struct{ *Resolver }
