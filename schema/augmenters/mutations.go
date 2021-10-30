package augmenters

import (
	"fmt"
	"strings"

	"github.com/spf13/cast"

	"github.com/jinzhu/inflection"

	"github.com/vektah/gqlparser/v2/ast"
)

type Mutations struct{}

func (m Mutations) DirectiveName() string {
	return "generateMutations"
}

func (m Mutations) Augment(s *ast.Schema) error {

	for _, v := range s.Types {
		d := v.Directives.ForName(m.DirectiveName())
		if d == nil {
			continue
		}
		def := s.Types[v.Name]
		// add mutation type if wasn't defined on schema
		if s.Mutation == nil {
			s.Mutation = &ast.Definition{
				Kind:        ast.Object,
				Description: "",
				Name:        "Mutation",
			}
			s.Types["Mutation"] = s.Mutation
		}

		args := d.ArgumentMap(nil)
		if c, ok := args["create"]; ok && cast.ToBool(c) {
			createFieldDef := m.addCreateMutation(s, def)
			s.Mutation.Fields = append(s.Mutation.Fields, createFieldDef)
		}
		if c, ok := args["delete"]; ok && cast.ToBool(c) {
			deleteFieldDef := m.addDeleteMutation(s, def)
			s.Mutation.Fields = append(s.Mutation.Fields, deleteFieldDef)

		}

	}
	return nil
}

func (m Mutations) addDeleteMutation(s *ast.Schema, obj *ast.Definition) *ast.FieldDefinition {
	deleteDef := &ast.FieldDefinition{
		Description: fmt.Sprintf("AutoGenerated input for %s", obj.Name),
		Name:        fmt.Sprintf("delete%s", inflection.Plural(obj.Name)),
		Arguments: []*ast.ArgumentDefinition{
			{
				Description: "cascade on delete",
				Name:        "cascade",
				Type: &ast.Type{
					NamedType: "Boolean",
				},
			},
		},
		Type: &ast.Type{
			NamedType: getPayloadObject(s, obj).Name,
		},
	}
	filterObj, ok := s.Types[fmt.Sprintf("%sFilterInput", obj.Name)]
	if ok {
		deleteDef.Arguments = append(deleteDef.Arguments, &ast.ArgumentDefinition{
			Description: "Filter objects to delete",
			Name:        "filter",
			Type:        &ast.Type{NamedType: filterObj.Name},
		})
	}
	return deleteDef
}

func (m Mutations) addCreateMutation(s *ast.Schema, obj *ast.Definition) *ast.FieldDefinition {

	inputObject := &ast.Definition{
		Kind:        ast.InputObject,
		Name:        fmt.Sprintf("Create%sInput", obj.Name),
		Description: fmt.Sprintf("AutoGenerated input for %s", obj.Name),
	}
	s.Types[fmt.Sprintf("create%s", inflection.Plural(obj.Name))] = inputObject
	for _, f := range obj.Fields {
		if strings.HasPrefix(f.Name, "__") {
			continue
		}
		fieldDef := s.Types[f.Type.Name()]
		// We don't support composite types
		if fieldDef.IsCompositeType() {
			continue
		}
		inputObject.Fields = append(inputObject.Fields, &ast.FieldDefinition{
			Name:        f.Name,
			Description: f.Description,
			Type:        f.Type,
		})
	}

	return &ast.FieldDefinition{
		Description: fmt.Sprintf("AutoGenerated input for %s", obj.Name),
		Name:        fmt.Sprintf("create%s", inflection.Plural(obj.Name)),
		Arguments: []*ast.ArgumentDefinition{
			{
				Description:  "",
				Name:         "inputs",
				DefaultValue: nil,
				Type: &ast.Type{
					Elem: &ast.Type{
						NamedType: fmt.Sprintf("Create%sInput", obj.Name),
						NonNull:   true,
					},
					NonNull: true,
				},
				Directives: nil,
				Position:   nil,
			},
		},
		Type: &ast.Type{
			NamedType: getPayloadObject(s, obj).Name,
		},
	}
}

func getPayloadObject(s *ast.Schema, obj *ast.Definition) *ast.Definition {
	payloadObjectName := fmt.Sprintf("%sPayload", inflection.Plural(obj.Name))
	if payloadObject, ok := s.Types[payloadObjectName]; ok {
		return payloadObject
	}
	payloadObject := &ast.Definition{
		Kind:        ast.Object,
		Description: "Autogenerated payload object",
		Name:        payloadObjectName,
		Fields: []*ast.FieldDefinition{
			{
				Description: "rows affection by mutation",
				Name:        "rows_affected",
				Type: &ast.Type{
					NamedType: "Int",
					NonNull:   true,
				},
			},
			{
				Description: obj.Description,
				Name:        inflection.Plural(strings.ToLower(obj.Name)),
				Type: &ast.Type{
					Elem: &ast.Type{NamedType: obj.Name},
				},
			}},
	}
	s.Types[payloadObjectName] = payloadObject
	return payloadObject
}
