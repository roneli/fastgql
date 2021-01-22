package augmenters

import (
	"fastgql/gql"
	"fmt"
	"github.com/spf13/cast"
	"github.com/vektah/gqlparser/v2/ast"
	"strings"
)


type Ordering struct {}

func (o Ordering) Name() string {
	return "generateArguments"
}

func (o Ordering) Augment(s *ast.Schema) error {
	for _, v := range s.Types {
		d := v.Directives.ForName(o.Name())
		if d == nil {
			continue
		}
		args := d.ArgumentMap(nil)
		if addOrdering, ok := args["ordering"]; ok && cast.ToBool(addOrdering) {
			o.addOrdering(s, v)
		}
	}
	return nil
}

func (o Ordering) addOrdering(s *ast.Schema, obj *ast.Definition) {
	for _, f := range obj.Fields {
		if strings.HasPrefix(f.Name, "__") {
			continue
		}
		if !gql.IsListType(f.Type) {
			continue
		}
		t := gql.GetType(f.Type)
		fieldDef, ok := s.Types[t.Name()]
		if !ok {
			continue
		}
		if !fieldDef.IsCompositeType() {
			continue
		}
		orderDef := o.buildOrderingEnum(s, fieldDef)
		if orderDef == nil {
			continue
		}
		// Finally we can add the argument
		f.Arguments = append(f.Arguments,
			&ast.ArgumentDefinition{
				Description: orderDef.Description,
				Name: "orderBy",
				Type: &ast.Type{NamedType: orderDef.Name},
			},
		)

	}
}

func (o Ordering) buildOrderingEnum(s *ast.Schema, obj *ast.Definition) *ast.Definition {

	orderInputDef := &ast.Definition{
		Kind:        ast.InputObject,
		Description: fmt.Sprintf("Ordering for %s", obj.Name),
		Name:        fmt.Sprintf("%sOrdering", obj.Name),
	}

	for _, f := range obj.Fields {
		fieldDef := s.Types[f.Type.Name()]
		// Ordering only supports first level ordering
		if !fieldDef.IsLeafType() {
			continue
		}
		orderInputDef.Fields = append(orderInputDef.Fields, &ast.FieldDefinition{
			Description: fmt.Sprintf("Order %s by %s", obj.Name, f.Name),
			Name:        f.Name,
			Type:        &ast.Type{NamedType: "_OrderingTypes"},
		})
	}
	if len(orderInputDef.Fields) == 0 {
		return nil
	}
	// Add ordering type
	s.Types[orderInputDef.Name] = orderInputDef
	return orderInputDef
}

