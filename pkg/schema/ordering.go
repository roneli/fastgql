package schema

import (
	"fmt"
	"log"

	"github.com/spf13/cast"
	"github.com/vektah/gqlparser/v2/ast"
)

func OrderByAugmenter(s *ast.Schema) error {
	for _, v := range s.Query.Fields {
		d := v.Directives.ForName(generateDirectiveName)
		if d == nil {
			continue
		}
		if !IsListType(v.Type) {
			continue
		}
		log.Printf("adding ordering to field %s@%s\n", v.Name, s.Query.Name)
		args := d.ArgumentMap(nil)
		if p, ok := args["ordering"]; ok && cast.ToBool(p) {
			if err := addOrderByArgsToField(s, s.Query, v); err != nil {
				return err
			}
		}
		if recursive := cast.ToBool(args["recursive"]); recursive {
			if err := addRecursive(s, s.Types[GetType(v.Type).Name()], "orderBy", addOrderByArgsToField); err != nil {
				return err
			}
		}
	}
	return nil
}

func addOrderByArgsToField(s *ast.Schema, obj *ast.Definition, field *ast.FieldDefinition) error {
	if skipAugment(field, "orderBy") {
		return nil
	}
	t := GetType(field.Type)
	fieldDef, ok := s.Types[t.Name()]
	if !ok || !fieldDef.IsCompositeType() {
		return nil
	}
	orderDef := buildOrderingEnum(s, fieldDef)
	if orderDef == nil {
		log.Printf("ordering for field %s@%s already exists skipping\n", field.Name, obj.Name)
		return nil
	}
	s.Types[orderDef.Name] = orderDef
	log.Printf("adding ordering to field %s@%s\n", field.Name, obj.Name)
	// Finally, we can add the argument
	field.Arguments = append(field.Arguments,
		&ast.ArgumentDefinition{
			Description: orderDef.Description,
			Name:        "orderBy",
			Type:        &ast.Type{Elem: &ast.Type{NamedType: orderDef.Name}},
		},
	)
	return nil
}

func buildOrderingEnum(s *ast.Schema, obj *ast.Definition) *ast.Definition {
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
