package builders

import (
	"context"
)

const filterCtx filterContext = "filter_context"

type filterContext string

// FilterFieldContext holds filter information for a field.
type FilterFieldContext struct {
	Filters map[string]interface{}
}

// WithFieldFilterContext adds filter context to the context.
func WithFieldFilterContext(ctx context.Context, rc *FilterFieldContext) context.Context {
	return context.WithValue(ctx, filterCtx, rc)
}

// GetFieldFilterContext retrieves filter context from the context.
func GetFieldFilterContext(ctx context.Context) *FilterFieldContext {
	if val, ok := ctx.Value(filterCtx).(*FilterFieldContext); ok {
		return val
	}
	return nil
}
