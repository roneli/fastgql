package builders

import (
	"context"
	"fmt"
	"strings"

	"github.com/iancoleman/strcase"

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

type OperationType string

const (
	QueryOperation     OperationType = "query"
	AggregateOperation OperationType = "aggregate"
	InsertOperation    OperationType = "insert"
	DeleteOperation    OperationType = "delete"
	UnknownOperation   OperationType = "unknown"
)

type Field struct {
	// Original *ast.Field
	*ast.Field
	// Field sub selection, if field type is TypeScalar, selections will be empty
	Selections []Field
	// FieldType i.e Scalar/Relation/Aggregate/Object etc'
	FieldType fieldType
	// Arguments passed to this field if any.
	Arguments map[string]interface{}
	// TypeDefinition is the ast.Schema Type, this is saved for easier access
	TypeDefinition *ast.Definition
	// Parent field of this field
	Parent *Field
}

func NewField(parent *Field, field *ast.Field, schema *ast.Schema, args map[string]interface{}) Field {
	typeName := getTypeName(field)
	typeDef := schema.Types[typeName]
	return Field{
		Field:          field,
		Selections:     nil,
		FieldType:      parseFieldType(field, typeDef),
		Arguments:      args,
		TypeDefinition: typeDef,
		Parent:         parent,
	}
}

func parseFieldType(field *ast.Field, typeDef *ast.Definition) fieldType {
	switch {
	case strings.HasSuffix(field.Name, "Aggregate"):
		return TypeAggregate
	case typeDef.IsCompositeType():
		if d := field.Definition.Directives.ForName("relation"); d != nil {
			return TypeRelation
		}
		return TypeObject
	default:
		return TypeScalar
	}
}

func (f Field) ForName(name string) (Field, error) {
	for _, s := range f.Selections {
		if s.Name == name {
			return s, nil
		}
	}
	return Field{}, fmt.Errorf("field doesn't exist")
}

func GetFilterInput(s *ast.Schema, f *ast.Definition) *ast.Definition {
	return s.Types[fmt.Sprintf("%sFilterInput", f.Name)]
}

func (f Field) GetTypeName() string {
	return f.TypeDefinition.Name
}

func (f Field) Relation() *RelationDirective {
	return GetRelationDirective(f.Definition)
}

func (f Field) Table() *TableDirective {
	d := f.TypeDefinition.Directives.ForName("table")
	if d == nil {
		return nil
	}
	return &TableDirective{
		Name:    GetArgumentValue(d.Arguments, "name"),
		Schema:  GetArgumentValue(d.Arguments, "schema"),
		Dialect: GetArgumentValue(d.Arguments, "dialect"),
	}
}

func getTypeName(f *ast.Field) string {
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
	orderFields := make([]OrderField, 0)
	for k, v := range argMap {
		orderFields = append(orderFields, OrderField{
			Key:  k,
			Type: OrderingTypes(cast.ToString(v)),
		})
	}
	return orderFields
}

func CollectFields(ctx context.Context, schema *ast.Schema) Field {
	resCtx := graphql.GetFieldContext(ctx)
	opCtx := graphql.GetOperationContext(ctx)
	filterCtx := GetFieldFilterContext(ctx)
	args := resCtx.Field.ArgumentMap(graphql.GetOperationContext(ctx).Variables)
	if filterCtx != nil {
		if _, ok := args["filter"]; !ok {
			args["filter"] = filterCtx.Filters
		} else {
			for k, v := range filterCtx.Filters {
				args["filter"].(map[string]interface{})[k] = v
			}
		}
	}
	f := NewField(nil, resCtx.Field.Field, schema, args)
	f.Selections = collectFields(&f, schema, opCtx, make(map[string]bool))
	return f
}

func GetOperationType(ctx context.Context) OperationType {
	opCtx := graphql.GetOperationContext(ctx)
	if opCtx.Operation.Operation == "mutation" {
		sel := opCtx.Operation.SelectionSet[0]
		field := sel.(*ast.Field)
		switch {
		case strings.HasPrefix(field.Name, "delete"):
			return DeleteOperation
		case strings.HasPrefix(field.Name, "create"):
			return InsertOperation
		}
		return UnknownOperation
	}

	return QueryOperation
}

func GetAggregateField(parentField, aggField Field) Field {
	fieldName := strings.Split(aggField.Name, "Aggregate")[0][1:]
	f, _ := parentField.ForName(fieldName)
	return f
}

func CollectFromQuery(field *ast.Field, doc *ast.QueryDocument, variables map[string]interface{}, arguments map[string]interface{}) Field {

	// TODO: fix
	return Field{
		Field:     field,
		FieldType: TypeObject,
		Arguments: arguments,
	}
}

func collectFields(parent *Field, schema *ast.Schema, opCtx *graphql.OperationContext, visited map[string]bool) []Field {
	groupedFields := make([]Field, 0)

	for _, sel := range parent.Field.SelectionSet {
		switch sel := sel.(type) {
		case *ast.Field:
			if !shouldIncludeNode(sel.Directives, opCtx.Variables) {
				continue
			}
			selField := getOrCreateAndAppendField(&groupedFields, sel.Alias, sel.ObjectDefinition, func() Field {
				return NewField(parent, sel, schema, resolveArguments(sel, opCtx.Variables))
			})
			// Add filter fields for relation form different provider, so they are returned by builder query
			if selField.FieldType == TypeRelation && selField.Table().Dialect != parent.Table().Dialect {
				for _, relationField := range selField.Relation().Fields {
					groupedFields = append(groupedFields, Field{
						Field: &ast.Field{
							Name:             strcase.ToLowerCamel(relationField),
							ObjectDefinition: parent.ObjectDefinition,
						},
						FieldType: TypeScalar,
						Parent:    parent,
					})
				}
				// No need to add the original selection as it exists in a different source
				continue
			}
			if selField.Field.SelectionSet != nil {
				// Add any sub selections of this field
				selField.Selections = append(selField.Selections, collectFields(selField, schema, opCtx, map[string]bool{})...)
			}
		case *ast.InlineFragment:
			if !shouldIncludeNode(sel.Directives, opCtx.Variables) {
				continue
			}
			for _, childField := range collectFields(parent, schema, opCtx, visited) {
				f := getOrCreateAndAppendField(&groupedFields, childField.Name, childField.ObjectDefinition, func() Field { return childField })
				f.Selections = append(f.Selections, childField.Selections...)
			}

		case *ast.FragmentSpread:
			if !shouldIncludeNode(sel.Directives, opCtx.Variables) {
				continue
			}
			fragmentName := sel.Name
			if _, seen := visited[fragmentName]; seen {
				continue
			}
			visited[fragmentName] = true

			fragment := opCtx.Doc.Fragments.ForName(fragmentName)
			if fragment == nil {
				// should never happen, validator has already run
				panic(fmt.Errorf("missing fragment %s", fragmentName))
			}

			for _, childField := range collectFields(parent, schema, opCtx, visited) {
				f := getOrCreateAndAppendField(&groupedFields, childField.Name, childField.ObjectDefinition, func() Field { return childField })
				f.Selections = append(f.Selections, childField.Selections...)
			}
		default:
			panic(fmt.Errorf("unsupported %T", sel))
		}
	}

	return groupedFields
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

func resolveArguments(f *ast.Field, variables map[string]interface{}) map[string]interface{} {
	if f.Definition == nil {
		return variables
	}
	return f.ArgumentMap(variables)
}
