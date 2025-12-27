package schema

import (
	"fmt"
	"log"
	"sort"
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

// resolveScalarOrEnumComparator resolves comparators for scalar and enum types
func resolveScalarOrEnumComparator(s *ast.Schema, field *ast.FieldDefinition, fieldType *ast.Type) *ast.Definition {
	if IsListType(field.Type) {
		return s.Types[fmt.Sprintf("%sListComparator", fieldType.Name())]
	}
	return s.Types[fmt.Sprintf("%sComparator", fieldType.Name())]
}

// resolveJsonPathScalarOrEnumComparator resolves JSONPath-specific comparators for JSON fields
func resolveJsonPathScalarOrEnumComparator(s *ast.Schema, field *ast.FieldDefinition, fieldType *ast.Type) *ast.Definition {
	if IsListType(field.Type) {
		// For arrays, we might need JsonPath*ListComparator in the future
		// For now, return nil or handle differently
		return nil
	}
	// Return JsonPath*Comparator types
	comparatorName := fmt.Sprintf("JsonPath%sComparator", fieldType.Name())
	return s.Types[comparatorName]
}

// resolveObjectFilterInput resolves filter input for object types in regular (non-JSON) context
func resolveObjectFilterInput(s *ast.Schema, fieldType *ast.Type) *ast.Definition {
	return s.Types[fmt.Sprintf("%sFilterInput", fieldType.Name())]
}

// resolveJsonObjectFilterInput resolves filter input for object types in JSON context
// Recursively creates nested JSON filter inputs if they don't exist
func resolveJsonObjectFilterInput(s *ast.Schema, fieldType *ast.Type, def *ast.Definition) *ast.Definition {
	nestedFilterInputName := fmt.Sprintf("%sFilterInput", fieldType.Name())
	if _, exists := s.Types[nestedFilterInputName]; !exists {
		createJsonTypeFilterInput(s, def, nestedFilterInputName)
	}
	return s.Types[nestedFilterInputName]
}

// resolveFieldComparator resolves the appropriate comparator or filter input for a field type in regular context
// Returns nil if no comparator/filter input can be resolved
func resolveFieldComparator(s *ast.Schema, field *ast.FieldDefinition, fieldType *ast.Type, def *ast.Definition) *ast.Definition {
	switch def.Kind {
	case ast.Scalar, ast.Enum:
		return resolveScalarOrEnumComparator(s, field, fieldType)
	case ast.Object:
		return resolveObjectFilterInput(s, fieldType)
	case ast.Interface:
		return resolveObjectFilterInput(s, fieldType)
	}
	return nil
}

// addLogicalOperators adds AND, OR, and NOT logical operators to a filter input
func addLogicalOperators(input *ast.Definition, filterInputName string) {
	input.Fields = append(input.Fields, []*ast.FieldDefinition{
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
		if hasJsonDirective && (def.Kind == ast.Object || def.Kind == ast.Interface) {
			// For @json fields with object types, create a FilterInput for the JSON type
			// This allows filtering like: attributes: { color: { eq: "red" } }
			filterInputName := fmt.Sprintf("%sFilterInput", fieldType.Name())
			if _, exists := s.Types[filterInputName]; !exists {
				// Create the FilterInput for this JSON type
				createJsonTypeFilterInput(s, def, filterInputName)
			}
			fieldDef = s.Types[filterInputName]
		} else {
			// Use the shared resolver for regular fields
			fieldDef = resolveFieldComparator(s, field, fieldType, def)
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
		// Collect implementation types and sort them for deterministic output
		implTypes := make([]string, 0)
		for k, imps := range s.Implements {
			for _, d := range imps {
				if d.Name == object.Name {
					implTypes = append(implTypes, k)
				}
			}
		}
		sort.Strings(implTypes)
		for _, k := range implTypes {
			log.Printf("adding filter input for interface implementation %s\n", k)
			name := fmt.Sprintf("%sFilterInput", k)
			input.Fields = append(input.Fields, &ast.FieldDefinition{
				Name:       strcase.ToLowerCamel(k),
				Type:       &ast.Type{NamedType: name},
				Directives: []*ast.Directive{{Name: "isInterfaceFilter"}},
			})
		}
	}
	addLogicalOperators(input, input.Name)
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
		case ast.Scalar:
			// Use JsonPath comparators for JSON fields
			fieldDef = resolveJsonPathScalarOrEnumComparator(s, field, fieldType)
			// If JsonPath comparator doesn't exist (e.g., for unsupported types), fall back to standard
			if fieldDef == nil {
				fieldDef = resolveScalarOrEnumComparator(s, field, fieldType)
			}
		case ast.Enum:
			// For enums, use standard comparator (enums are just equality checks with ==)
			fieldDef = resolveScalarOrEnumComparator(s, field, fieldType)
		case ast.Object:
			fieldDef = resolveJsonObjectFilterInput(s, fieldType, def)
		}

		if fieldDef != nil {
			filterInput.Fields = append(filterInput.Fields, &ast.FieldDefinition{
				Name: field.Name,
				Type: &ast.Type{NamedType: fieldDef.Name},
			})
		}
	}

	// Add logical operators
	addLogicalOperators(filterInput, filterInputName)

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
