package builders

import (
	"context"
	"fmt"
	"strings"

	"github.com/99designs/gqlgen/graphql"
	"github.com/spf13/cast"
	"github.com/vektah/gqlparser/v2/ast"
)

type fieldType string

const (
	TypeScalar    fieldType = "Scalar"
	TypeRelation  fieldType = "Relation"
	TypeAggregate fieldType = "Aggregate"
	TypeObject    fieldType = "Object"
)

type Field struct {
	*ast.Field
	Selections []Field
	FieldType  fieldType
	Arguments  map[string]interface{}
}

func (f Field) GetTypeName() string {
	typeName := f.Definition.Type.Name()
	if strings.HasSuffix(f.Name, "Aggregate") {
		originalFieldName := strings.Split(f.Name, "Aggregate")[0][1:]
		typeName = f.ObjectDefinition.Fields.ForName(originalFieldName).Type.Name()
	}
	return typeName
}

func CollectOrdering(ordering interface{}) ([]OrderField, error) {
	switch orderings := ordering.(type) {
	case map[string]interface{}:
		return buildOrderingHelper(orderings), nil
	case []interface{}:
		var orderFields []OrderField
		for _, o := range orderings {
			argMap, ok := o.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("invalid value type")
			}
			orderFields = append(orderFields, buildOrderingHelper(argMap)...)
		}
		return orderFields, nil
	default:
		panic(fmt.Sprintf("unknown ordering type %v", orderings))
	}
}

func buildOrderingHelper(argMap map[string]interface{}) []OrderField {
	orderFields := make([]OrderField, len(argMap))
	for k, v := range argMap {
		orderFields = append(orderFields, OrderField{
			Key:  k,
			Type: OrderingTypes(cast.ToString(v)),
		})
	}
	return orderFields
}

func CollectFields(ctx context.Context) Field {
	resCtx := graphql.GetFieldContext(ctx)
	return Field{
		resCtx.Field.Field,
		collectFields(graphql.GetOperationContext(ctx), resCtx.Field.Selections, map[string]bool{}),
		TypeObject,
		resCtx.Field.ArgumentMap(graphql.GetOperationContext(ctx).Variables),
	}
}

func collectFields(reqCtx *graphql.OperationContext, selSet ast.SelectionSet, visited map[string]bool) []Field {
	groupedFields := make([]Field, 0, len(selSet))

	for _, sel := range selSet {
		switch sel := sel.(type) {
		case *ast.Field:
			if !shouldIncludeNode(sel.Directives, reqCtx.Variables) {
				continue
			}
			f := getOrCreateAndAppendField(&groupedFields, sel.Alias, sel.ObjectDefinition, func() Field {
				return collectField(reqCtx, sel)
			})
			f.Selections = append(f.Selections, collectFields(reqCtx, sel.SelectionSet, map[string]bool{})...)
		case *ast.InlineFragment:
			if !shouldIncludeNode(sel.Directives, reqCtx.Variables) {
				continue
			}
			for _, childField := range collectFields(reqCtx, sel.SelectionSet, visited) {
				f := getOrCreateAndAppendField(&groupedFields, childField.Name, childField.ObjectDefinition, func() Field { return childField })
				f.Selections = append(f.Selections, childField.Selections...)
			}

		case *ast.FragmentSpread:
			if !shouldIncludeNode(sel.Directives, reqCtx.Variables) {
				continue
			}
			fragmentName := sel.Name
			if _, seen := visited[fragmentName]; seen {
				continue
			}
			visited[fragmentName] = true

			fragment := reqCtx.Doc.Fragments.ForName(fragmentName)
			if fragment == nil {
				// should never happen, validator has already run
				panic(fmt.Errorf("missing fragment %s", fragmentName))
			}

			for _, childField := range collectFields(reqCtx, fragment.SelectionSet, visited) {
				f := getOrCreateAndAppendField(&groupedFields, childField.Name, childField.ObjectDefinition, func() Field { return childField })
				f.Selections = append(f.Selections, childField.Selections...)
			}
		default:
			panic(fmt.Errorf("unsupported %T", sel))
		}
	}

	return groupedFields
}

func collectField(reqCtx *graphql.OperationContext, f *ast.Field) Field {
	if strings.HasSuffix(f.Name, "Aggregate") {
		return Field{f, nil, TypeAggregate, f.ArgumentMap(reqCtx.Variables)}
	}
	// check if relational object
	if d := f.Definition.Directives.ForName("sqlRelation"); d != nil {
		return Field{f, nil, TypeRelation, f.ArgumentMap(reqCtx.Variables)}
	}
	return Field{f, nil, TypeScalar, f.ArgumentMap(reqCtx.Variables)}
}

func getOrCreateAndAppendField(c *[]Field, name string, objectDefinition *ast.Definition, creator func() Field) *Field {
	for i, cf := range *c {
		if cf.Alias == name && (cf.ObjectDefinition == objectDefinition || (cf.ObjectDefinition != nil && objectDefinition != nil && cf.ObjectDefinition.Name == objectDefinition.Name)) {
			return &(*c)[i]
		}
	}

	f := creator()

	*c = append(*c, f)
	return &(*c)[len(*c)-1]
}

func shouldIncludeNode(directives ast.DirectiveList, variables map[string]interface{}) bool {
	if len(directives) == 0 {
		return true
	}

	skip, include := false, true

	if d := directives.ForName("skip"); d != nil {
		skip = resolveIfArgument(d, variables)
	}

	if d := directives.ForName("include"); d != nil {
		include = resolveIfArgument(d, variables)
	}

	return !skip && include
}

func resolveIfArgument(d *ast.Directive, variables map[string]interface{}) bool {
	arg := d.Arguments.ForName("if")
	if arg == nil {
		panic(fmt.Sprintf("%s: argument 'if' not defined", d.Name))
	}
	value, err := arg.Value.Value(variables)
	if err != nil {
		panic(err)
	}
	ret, ok := value.(bool)
	if !ok {
		panic(fmt.Sprintf("%s: argument 'if' is not a boolean", d.Name))
	}
	return ret
}
