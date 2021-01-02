package augmenters

import (
	"fastgql/gql"
	"fmt"
"github.com/spf13/cast"
"github.com/vektah/gqlparser/v2/ast"
	"strings"
)


type createdInputDef struct {
	object *ast.Definition
	input *ast.Definition
}

type FilterInput struct {}

func (f FilterInput) Name() string {
	return "generateFilterInput"
}

func (f FilterInput) Augment(s *ast.Schema) error {
	inputs := f.initInputs(s)
	for _, input := range inputs {
		f.buildFilterInput(s, input.input, input.object)
	}
	return nil
}

func (f FilterInput) buildFilterInput(s *ast.Schema, input *ast.Definition, object *ast.Definition) {

	for _, f := range object.Fields {
		fieldType := gql.GetType(f.Type)
		def, ok  := s.Types[fieldType.Name()]
		if !ok {
			continue
		}
		var fieldDef *ast.Definition
		switch def.Kind {
		case ast.Scalar, ast.Enum:
			if gql.IsListType(f.Type) {
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
			Name: fmt.Sprintf("%s", f.Name),
			Type: &ast.Type{NamedType: fieldDef.Name},
		})
	}
	input.Fields = append(input.Fields, []*ast.FieldDefinition{
		{
			Name: "AND",
			Description: "Logical AND of FilterInput",
			Type: &ast.Type{
				Elem: &ast.Type{
					NamedType: input.Name,
				},
			},
		},
		{
			Name: "OR",
			Description: "Logical OR of FilterInput",
			Type: &ast.Type{
				Elem: &ast.Type{
					NamedType: input.Name,
				},
			},
		},
		{
			Name: "NOT",
			Description: "Logical NOT of FilterInput",
			Type: &ast.Type{
				NamedType: input.Name,
			},
		},
	}...)
}


// initInputs initialize all filter inputs before adding fields to avoid recursive reference
func (f FilterInput) initInputs(s *ast.Schema) []*createdInputDef {

	defs := make([]*createdInputDef, 0)
	for _, obj := range s.Types {
		// Check if object has a generateFilterInput directive
		d := obj.Directives.ForName(f.Name())
		if d == nil {
			continue
		}
		args := d.ArgumentMap(nil)
		name := cast.ToString(args["name"])
		s.Types[name] = &ast.Definition{
			Kind:        ast.InputObject,
			Description: cast.ToString(args["description"]),
			Name:        name,
		}
		defs = append(defs, &createdInputDef{obj, s.Types[name]})
	}
	return defs
}


type FilterArguments struct {}

func (f FilterArguments) Name() string {
	return "generateArguments"
}

func (f FilterArguments) Augment(s *ast.Schema) error {
	for _, v := range s.Types {
		d := v.Directives.ForName(f.Name())
		if d == nil {
			continue
		}
		args := d.ArgumentMap(nil)
		if addPagination, ok := args["filter"]; ok && cast.ToBool(addPagination) {
			f.addFilter(s, v)
		}
	}
	return nil
}

func (f FilterArguments) addFilter(s *ast.Schema, obj *ast.Definition) {
	for _, f := range obj.Fields {
		if strings.HasPrefix(f.Name, "__") {
			continue
		}
		input, ok:= s.Types[fmt.Sprintf("%sFilterInput", f.Type.Name())]
		if !ok {
			continue
		}
		
		f.Arguments = append(f.Arguments,
			&ast.ArgumentDefinition{Description: fmt.Sprintf("Filter %s", f.Name),
				Name:         "filter",
				Type:         &ast.Type{NamedType: input.Name},
			},
		)
	}
}
