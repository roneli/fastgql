package augmenters

import (
	"fmt"
	"github.com/roneli/fastgql/pkg/schema/gql"
	"log"

	"github.com/spf13/cast"
	"github.com/vektah/gqlparser/v2/ast"
)

type Ordering struct{}

func (o Ordering) DirectiveName() string {
	return "generate"
}

func (o Ordering) Name() string {
	return "ordering"
}

func (o Ordering) Augment(s *ast.Schema) error {
	for _, v := range s.Types {
		d := v.Directives.ForName(o.DirectiveName())
		if d == nil {
			continue
		}
		args := d.ArgumentMap(nil)
		recursive := cast.ToBool(args["recursive"])
		if addOrdering, ok := args["ordering"]; ok && cast.ToBool(addOrdering) {
			o.addOrdering(s, v, nil, recursive)
		}
	}
	return nil
}

func (o Ordering) addOrdering(s *ast.Schema, obj *ast.Definition, parent *ast.Definition, recursive bool) {
	for _, f := range obj.Fields {
		// avoid recurse and adding to internal objects
		if skipAugment(f, "orderBy") {
			continue
		}
		fieldType := s.Types[f.Type.Name()]
		if gql.IsScalarListType(s, f.Type) || !gql.IsListType(f.Type) {
			if recursive && fieldType.IsCompositeType() && parent != fieldType {
				o.addOrdering(s, fieldType, obj, recursive)
			}
			continue
		}
		t := gql.GetType(f.Type)
		fieldDef, ok := s.Types[t.Name()]
		if !ok || !fieldDef.IsCompositeType() {
			continue
		}
		orderDef := o.buildOrderingEnum(s, fieldDef)
		if orderDef == nil {
			log.Printf("ordering for field %s@%s already exists skipping\n", f.Name, obj.Name)
			continue
		}
		log.Printf("adding ordering to field %s@%s\n", f.Name, obj.Name)
		// Finally, we can add the argument
		f.Arguments = append(f.Arguments,
			&ast.ArgumentDefinition{
				Description: orderDef.Description,
				Name:        "orderBy",
				Type:        &ast.Type{Elem: &ast.Type{NamedType: orderDef.Name}},
			},
		)
		if recursive && fieldType.IsCompositeType() {
			log.Printf("adding ordering to field %s@%s\n", f.Name, obj.Name)
			o.addOrdering(s, fieldType, obj, recursive)
		}
	}
}

func (o Ordering) buildOrderingEnum(s *ast.Schema, obj *ast.Definition) *ast.Definition {

	orderInputDef := &ast.Definition{
		Kind:        ast.InputObject,
		Description: fmt.Sprintf("Ordering for %s", obj.Name),
		Name:        fmt.Sprintf("%sOrdering", obj.Name),
	}
	log.Printf("adding ordering for %s\n", obj.Name)
	for _, f := range obj.Fields {
		fieldDef := s.Types[f.Type.Name()]
		// Ordering only supports first level ordering
		if !fieldDef.IsLeafType() {
			continue
		}
		log.Printf("adding order field %s for %s\n", f.Name, obj.Name)
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
