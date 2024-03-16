package builders

import (
	"context"
	"reflect"

	"github.com/roneli/fastgql/pkg/schema"

	"github.com/vektah/gqlparser/v2/ast"

	"github.com/iancoleman/strcase"
)

const filterCtx FilterContext = "filter_context"

type FilterContext string

type FilterFieldContext struct {
	Filters map[string]interface{}
}

func AddRelationFilters(ctx context.Context, s *ast.Schema, obj interface{}) context.Context {
	// TODO: Use collect field
	field := CollectFields(ctx, s)
	relation := schema.GetRelationDirective(field.ObjectDefinition.Fields.ForName(field.Name))
	filters := make(map[string]interface{}, len(relation.Fields))
	for i := 0; i < len(relation.Fields); i++ {
		fieldValue := getFieldValue(obj, strcase.ToLowerCamel(relation.Fields[i]))
		filters[relation.References[i]] = map[string]interface{}{"eq": fieldValue}
	}
	return WithFieldFilterContext(ctx, &FilterFieldContext{Filters: filters})
}

func WithFieldFilterContext(ctx context.Context, rc *FilterFieldContext) context.Context {
	return context.WithValue(ctx, filterCtx, rc)
}

func GetFieldFilterContext(ctx context.Context) *FilterFieldContext {
	if val, ok := ctx.Value(filterCtx).(*FilterFieldContext); ok {
		return val
	}
	return nil
}

func getFieldValue(obj interface{}, fieldName string) interface{} {
	t := reflect.TypeOf(obj)
	v := reflect.ValueOf(obj)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
		v = v.Elem()
	}
	n := t.NumField()
	for i := 0; i < n; i++ {
		ft := t.Field(i)
		if ft.Name == fieldName {
			return v.Field(i).Interface()
		}
		if ft.Tag.Get("json") == fieldName || ft.Tag.Get("db") == fieldName {
			return v.Field(i).Interface()
		}
	}
	return nil
}
