package augmenters

import (
	"fmt"
	"log"
	"strings"

	"github.com/roneli/fastgql/pkg/schema/gql"

	"github.com/spf13/cast"
	"github.com/vektah/gqlparser/v2/ast"
)

type createdInputDef struct {
	object *ast.Definition
	input  *ast.Definition
}

type FilterInput struct{}

func (f FilterInput) DirectiveName() string {
	return "generateFilterInput"
}

func (f FilterInput) Name() string {
	return "filterInput"
}

func (f FilterInput) Augment(s *ast.Schema) error {
	inputs := f.initInputs(s)
	for _, input := range inputs {
		f.buildFilterInput(s, input.input, input.object)
	}
	return nil
}

func (f FilterInput) buildFilterInput(s *ast.Schema, input *ast.Definition, object *ast.Definition) {
	for _, field := range object.Fields {
		fieldType := gql.GetType(field.Type)
		def, ok := s.Types[fieldType.Name()]
		if !ok {
			continue
		}
		var fieldDef *ast.Definition
		switch def.Kind {
		case ast.Scalar, ast.Enum:
			if gql.IsListType(field.Type) {
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
func (f FilterInput) initInputs(s *ast.Schema) []*createdInputDef {
	defs := make([]*createdInputDef, 0)
	for _, obj := range s.Types {
		// Check if object has a generateFilterInput directive
		d := obj.Directives.ForName(f.DirectiveName())
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

type FilterArguments struct{}

func (fa FilterArguments) Name() string {
	return "filterArguments"
}

func (fa FilterArguments) DirectiveName() string {
	return "generate"
}

func (fa FilterArguments) Augment(s *ast.Schema) error {
	for _, v := range s.Types {
		d := v.Directives.ForName(fa.DirectiveName())
		if d == nil {
			continue
		}

		args := d.ArgumentMap(nil)
		recursive := cast.ToBool(args["recursive"])
		if addFilters, ok := args["filter"]; ok && cast.ToBool(addFilters) {
			log.Printf("adding filter arguments for %s\n", v.Name)
			fa.addFilter(s, v, nil, recursive)
		}
	}
	return nil
}

func (fa FilterArguments) addFilter(s *ast.Schema, obj *ast.Definition, parent *ast.Definition, recursive bool) {
	for _, f := range obj.Fields {
		// avoid recurse and Skip "special" field types such as type name etc'
		if skipAugment(f, "filter") {
			log.Printf("skipping adding filter for field %s\n", f.Name)
			continue
		}
		fieldType := s.Types[f.Type.Name()]
		if gql.IsScalarListType(s, f.Type) {
			if recursive && fieldType.IsCompositeType() && fieldType != parent {
				fa.addFilter(s, fieldType, obj, recursive)
			}
			continue
		}
		if fieldType.IsLeafType() {
			continue
		}
		log.Printf("building adding filters for field %s\n", f.Name)
		var typeName string
		if strings.HasSuffix(f.Name, "Aggregate") {
			fieldName := strings.Split(f.Name, "Aggregate")[0][1:]
			fieldDef := obj.Fields.ForName(fieldName)
			if fieldDef == nil {
				continue
			}
			typeName = fmt.Sprintf("%sFilterInput", fieldDef.Type.Name())
		} else {
			typeName = fmt.Sprintf("%sFilterInput", f.Type.Name())
		}

		input, ok := s.Types[typeName]
		if !ok {
			continue
		}
		log.Printf("adding filter argument for field %s\n", f.Name)
		f.Arguments = append(f.Arguments,
			&ast.ArgumentDefinition{Description: fmt.Sprintf("Filter %s", f.Name),
				Name: "filter",
				Type: &ast.Type{NamedType: input.Name},
			},
		)
		if recursive && fieldType.IsCompositeType() {
			log.Printf("adding ordering to field %s@%s\n", f.Name, obj.Name)
			fa.addFilter(s, fieldType, obj, recursive)
		}
	}
}
