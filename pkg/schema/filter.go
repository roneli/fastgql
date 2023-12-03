package schema

import (
	"fmt"
	"github.com/spf13/cast"
	"github.com/vektah/gqlparser/v2/ast"
	"log"
	"strings"
)

const filterInputDirectiveName = "generateFilterInput"

type createdInputDef struct {
	object *ast.Definition
	input  *ast.Definition
}

func FilterArgAugmenter(s *ast.Schema) error {
	for _, v := range append(s.Query.Fields, s.Mutation.Fields...) {
		d := v.Directives.ForName("generate")
		if d == nil {
			continue
		}
		if !IsListType(v.Type) {
			continue
		}
		log.Printf("adding filter to field %s@%s\n", v.Name, s.Query.Name)
		args := d.ArgumentMap(nil)
		if p, ok := args["filter"]; ok && cast.ToBool(p) {
			if err := addFilterToFieldArgs(s, s.Query, v); err != nil {
				return err
			}
		}
		if recursive := cast.ToBool(args["recursive"]); recursive {
			if err := addRecursive(s, s.Types[GetType(v.Type).Name()], "filter", addFilterToFieldArgs); err != nil {
				return err
			}
		}
	}
	return nil
}

func FilterInputAugmenter(s *ast.Schema) error {
	inputs := initInputs(s)
	for _, input := range inputs {
		buildFilterInput(s, input.input, input.object)
	}
	return nil
}

func addFilterToFieldArgs(s *ast.Schema, obj *ast.Definition, field *ast.FieldDefinition) error {
	if skipAugment(field, "filter") {
		return nil
	}
	typeName := fmt.Sprintf("%sFilterInput", field.Type.Name())
	if strings.HasSuffix(field.Name, "Aggregate") {
		fieldName := strings.Split(field.Name, "Aggregate")[0][1:]
		fieldDef := obj.Fields.ForName(fieldName)
		if fieldDef == nil {
			return nil
		}
		typeName = fmt.Sprintf("%sFilterInput", fieldDef.Type.Name())
	}
	input, ok := s.Types[typeName]
	if !ok {
		return nil
	}
	log.Printf("adding filter argument for field %s\n", field.Name)
	field.Arguments = append(field.Arguments,
		&ast.ArgumentDefinition{Description: fmt.Sprintf("Filter %s", field.Name),
			Name: "filter",
			Type: &ast.Type{NamedType: input.Name},
		},
	)
	return nil
}

func buildFilterInput(s *ast.Schema, input *ast.Definition, object *ast.Definition) {
	for _, field := range object.Fields {
		fieldType := GetType(field.Type)
		def, ok := s.Types[fieldType.Name()]
		if !ok {
			continue
		}
		var fieldDef *ast.Definition
		switch def.Kind {
		case ast.Scalar, ast.Enum:
			if IsListType(field.Type) {
				fieldDef = s.Types[fmt.Sprintf("%sListComparator", fieldType.Name())]
			} else {
				fieldDef = s.Types[fmt.Sprintf("%sComparator", fieldType.Name())]
			}
		case ast.Object, ast.Interface:
			fieldDef = s.Types[fmt.Sprintf("%sFilterInput", fieldType.Name())]
		}

		if fieldDef == nil {
			continue
		}

		input.Fields = append(input.Fields, &ast.FieldDefinition{
			Name: field.Name,
			Type: &ast.Type{NamedType: fieldDef.Name},
		})
	}
	input.Fields = append(input.Fields, []*ast.FieldDefinition{
		{
			Name:        "AND",
			Description: "Logical AND of FilterInput",
			Type: &ast.Type{
				Elem: &ast.Type{
					NamedType: input.Name,
				},
			},
		},
		{
			Name:        "OR",
			Description: "Logical OR of FilterInput",
			Type: &ast.Type{
				Elem: &ast.Type{
					NamedType: input.Name,
				},
			},
		},
		{
			Name:        "NOT",
			Description: "Logical NOT of FilterInput",
			Type: &ast.Type{
				NamedType: input.Name,
			},
		},
	}...)
}

// initInputs initialize all filter inputs before adding fields to avoid recursive reference
func initInputs(s *ast.Schema) []*createdInputDef {
	defs := make([]*createdInputDef, 0)
	for _, obj := range s.Types {
		// Check if object has a generateFilterInput directive
		d := obj.Directives.ForName(filterInputDirectiveName)
		if d == nil {
			continue
		}
		args := d.ArgumentMap(nil)
		name := fmt.Sprintf("%sFilterInput", obj.Name)
		s.Types[name] = &ast.Definition{
			Kind:        ast.InputObject,
			Description: cast.ToString(args["description"]),
			Name:        name,
		}
		defs = append(defs, &createdInputDef{obj, s.Types[name]})
	}
	return defs
}
