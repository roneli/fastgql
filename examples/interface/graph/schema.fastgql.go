package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"

	"github.com/roneli/fastgql/examples/interface/graph/generated"
	"github.com/roneli/fastgql/examples/interface/graph/model"
)

func (r *queryResolver) Person(ctx context.Context) (*model.Person, error) {
	var data *model.Person
	if err := r.Executor.Scan(ctx, "postgres", &data); err != nil {
		return nil, err
	}
	return data, nil
}

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type queryResolver struct{ *Resolver }
