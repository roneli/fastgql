package schema

import (
	"fmt"
	"log"
	"strings"

	"github.com/iancoleman/strcase"

	"github.com/spf13/cast"
	"github.com/vektah/gqlparser/v2/ast"
)

const filterInputDirectiveName = "generateFilterInput"

type createdInputDef struct {
	object *ast.Definition
	input  *ast.Definition
}

func FilterArgAugmenter(s *ast.Schema) error {
	for _, v := range s.Query.Fields {
		d := v.Directives.ForName(generateDirectiveName)
		if d == nil {
			continue
		}
		log.Printf("adding filter to field %s@%s\n", v.Name, s.Query.Name)
		args := d.ArgumentMap(nil)
		if p, ok := args["filter"]; ok && cast.ToBool(p) {
			if err := addFilterToQueryFieldArgs(s, s.Query, v); err != nil {
				return err
			}
		}
		if recursive := cast.ToBool(args["recursive"]); recursive {
			if err := addRecursive(s, s.Types[GetType(v.Type).Name()], "filter", addFilterToQueryFieldArgs); err != nil {
				return err
			}
		}
	}
	if s.Mutation == nil {
		return nil
	}
	// add filter to mutation fields
	for _, v := range s.Mutation.Fields {
		d := v.Directives.ForName(generateDirectiveName)
		if d == nil {
			continue
		}
		log.Printf("adding filter to mutation field %s@%s\n", v.Name, s.Mutation.Name)
		args := d.ArgumentMap(nil)
		if p, ok := args["filter"]; ok && cast.ToBool(p) {
			if err := addFilterToMutationField(s, v, cast.ToString(args["filterTypeName"])); err != nil {
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

func addFilterToMutationField(s *ast.Schema, field *ast.FieldDefinition, filterTypeName string) error {
	if skipAugment(field, "filter") {
		return nil
	}
	input, ok := s.Types[filterTypeName]
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

func addFilterToQueryFieldArgs(s *ast.Schema, obj *ast.Definition, field *ast.FieldDefinition) error {
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

		// Check if field has @json directive
		hasJsonDirective := field.Directives.ForName("json") != nil

		var fieldDef *ast.Definition
		switch def.Kind {
		case ast.Scalar, ast.Enum:
			if IsListType(field.Type) {
				fieldDef = s.Types[fmt.Sprintf("%sListComparator", fieldType.Name())]
			} else {
				fieldDef = s.Types[fmt.Sprintf("%sComparator", fieldType.Name())]
			}
		case ast.Object, ast.Interface:
			if hasJsonDirective {
				// For @json fields with object types, create a FilterInput for the JSON type
				// This allows filtering like: attributes: { color: { eq: "red" } }
				filterInputName := fmt.Sprintf("%sFilterInput", fieldType.Name())
				if _, exists := s.Types[filterInputName]; !exists {
					// Create the FilterInput for this JSON type
					createJsonTypeFilterInput(s, def, filterInputName)
				}
				fieldDef = s.Types[filterInputName]
			} else {
				// Regular object/interface relation
				fieldDef = s.Types[fmt.Sprintf("%sFilterInput", fieldType.Name())]
			}
		}

		if fieldDef == nil {
			continue
		}

		input.Fields = append(input.Fields, &ast.FieldDefinition{
			Name: field.Name,
			Type: &ast.Type{NamedType: fieldDef.Name},
		})
	}
	// if object is an interface, we need to create a filter input for each of its implementations
	if object.IsAbstractType() {
		log.Printf("adding filter input for interface %s\n", object.Name)
		for k, imps := range s.Implements {
			for _, d := range imps {
				if d.Name == object.Name {
					log.Printf("adding filter input for interface implementation %s\n", k)
					name := fmt.Sprintf("%sFilterInput", k)
					input.Fields = append(input.Fields, &ast.FieldDefinition{
						Name:       strcase.ToLowerCamel(k),
						Type:       &ast.Type{NamedType: name},
						Directives: []*ast.Directive{{Name: "isInterfaceFilter"}},
					})

				}
			}
		}
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

// createJsonTypeFilterInput creates a FilterInput for a JSON object type
// This allows typed JSON fields to use the same filter structure as relations
func createJsonTypeFilterInput(s *ast.Schema, jsonType *ast.Definition, filterInputName string) {
	log.Printf("creating filter input for JSON type %s\n", jsonType.Name)

	filterInput := &ast.Definition{
		Kind:        ast.InputObject,
		Description: fmt.Sprintf("Filter input for JSON type %s", jsonType.Name),
		Name:        filterInputName,
		Fields:      make([]*ast.FieldDefinition, 0),
	}

	// Add field comparators for the JSON type fields
	for _, field := range jsonType.Fields {
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
		case ast.Object:
			// Nested JSON object - recursively create its filter input
			nestedFilterInputName := fmt.Sprintf("%sFilterInput", fieldType.Name())
			if _, exists := s.Types[nestedFilterInputName]; !exists {
				createJsonTypeFilterInput(s, def, nestedFilterInputName)
			}
			fieldDef = s.Types[nestedFilterInputName]
		}

		if fieldDef != nil {
			filterInput.Fields = append(filterInput.Fields, &ast.FieldDefinition{
				Name: field.Name,
				Type: &ast.Type{NamedType: fieldDef.Name},
			})
		}
	}

	// Add logical operators
	filterInput.Fields = append(filterInput.Fields, []*ast.FieldDefinition{
		{
			Name:        "AND",
			Description: "Logical AND of FilterInput",
			Type: &ast.Type{
				Elem: &ast.Type{
					NamedType: filterInputName,
				},
			},
		},
		{
			Name:        "OR",
			Description: "Logical OR of FilterInput",
			Type: &ast.Type{
				Elem: &ast.Type{
					NamedType: filterInputName,
				},
			},
		},
		{
			Name:        "NOT",
			Description: "Logical NOT of FilterInput",
			Type: &ast.Type{
				NamedType: filterInputName,
			},
		},
	}...)

	s.Types[filterInputName] = filterInput
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
		// if object is an interface, we need to create a filter input for each of its implementations
		if obj.IsAbstractType() {
			log.Printf("adding filter input for interface %s\n", obj.Name)
			for k, imps := range s.Implements {
				for _, d := range imps {
					if d.Name == obj.Name {
						log.Printf("adding filter input for interface implementation %s\n", k)
						name := fmt.Sprintf("%sFilterInput", k)
						s.Types[name] = &ast.Definition{
							Kind:        ast.InputObject,
							Description: cast.ToString(args["description"]),
							Name:        name,
						}
						defs = append(defs, &createdInputDef{s.Types[k], s.Types[name]})
					}
				}
			}
		}
	}
	return defs
}
