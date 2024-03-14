package schema

import (
	"fmt"
	"log"
	"strings"

	"github.com/jinzhu/inflection"
	"github.com/spf13/cast"
	"github.com/vektah/gqlparser/v2/ast"
)

const mutationsDirectiveName = "generateMutations"

func MutationsAugmenter(s *ast.Schema) error {
	if !schemaHasMutationDirective(s) {
		return nil
	}
	// add mutation type if wasn't defined on schema
	if s.Mutation == nil {
		s.Mutation = &ast.Definition{
			Kind:        ast.Object,
			Description: "Graphql Mutations",
			Name:        "Mutation",
		}
		s.Types["Mutation"] = s.Mutation
	}
	for _, v := range s.Types {
		d := v.Directives.ForName(mutationsDirectiveName)
		if d == nil {
			log.Printf("skipping %s, no directive %s", v.Name, mutationsDirectiveName)
			continue
		}
		log.Printf("adding mutations for %s", v.Name)
		def := s.Types[v.Name]
		args := d.ArgumentMap(nil)
		if c, ok := args["create"]; ok && cast.ToBool(c) {
			createFieldDef := addCreateMutation(s, def)
			s.Mutation.Fields = append(s.Mutation.Fields, createFieldDef)
		}
		if c, ok := args["delete"]; ok && cast.ToBool(c) {
			deleteFieldDef := addDeleteMutation(s, def)
			s.Mutation.Fields = append(s.Mutation.Fields, deleteFieldDef)
		}
		if c, ok := args["update"]; ok && cast.ToBool(c) {
			s.Mutation.Fields = append(s.Mutation.Fields, addUpdateMutation(s, def))
		}
	}
	return nil
}

func schemaHasMutationDirective(s *ast.Schema) bool {
	for _, v := range s.Types {
		if v.Directives.ForName(mutationsDirectiveName) != nil {
			return true
		}
	}
	return false
}

func addDeleteMutation(s *ast.Schema, obj *ast.Definition) *ast.FieldDefinition {
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
		Directives: []*ast.Directive{
			{
				Name: generateDirectiveName,
				Arguments: []*ast.Argument{
					{
						Name:  "filter",
						Value: &ast.Value{Kind: ast.BooleanValue, Raw: "true"},
					},
					{
						Name: "filterTypeName",
						Value: &ast.Value{
							Kind: ast.StringValue,
							Raw:  fmt.Sprintf("%sFilterInput", obj.Name),
						},
					},
				},
				Definition: s.Directives[generateDirectiveName],
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

func addCreateMutation(s *ast.Schema, obj *ast.Definition) *ast.FieldDefinition {
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

func addUpdateMutation(s *ast.Schema, obj *ast.Definition) *ast.FieldDefinition {
	inputObject := &ast.Definition{
		Kind:        ast.InputObject,
		Name:        fmt.Sprintf("Update%sInput", obj.Name),
		Description: fmt.Sprintf("AutoGenerated update input for %s", obj.Name),
	}
	s.Types[fmt.Sprintf("update%s", inflection.Plural(obj.Name))] = inputObject
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
			Type: &ast.Type{
				NamedType: f.Type.NamedType,
				NonNull:   false,
			},
		})
	}

	updateDef := &ast.FieldDefinition{
		Description: fmt.Sprintf("AutoGenerated input for %s", obj.Name),
		Name:        fmt.Sprintf("update%s", inflection.Plural(obj.Name)),
		Directives: []*ast.Directive{
			{
				Name: generateDirectiveName,
				Arguments: []*ast.Argument{
					{
						Name:  "filter",
						Value: &ast.Value{Kind: ast.BooleanValue, Raw: "true"},
					},
					{
						Name: "filterTypeName",
						Value: &ast.Value{
							Kind: ast.StringValue,
							Raw:  fmt.Sprintf("%sFilterInput", obj.Name),
						},
					},
				},
				Definition: s.Directives[generateDirectiveName],
			},
		},
		Arguments: []*ast.ArgumentDefinition{
			{
				Description:  "",
				Name:         "input",
				DefaultValue: nil,
				Type: &ast.Type{
					NamedType: fmt.Sprintf("Update%sInput", obj.Name),
					NonNull:   true,
				},
				Directives: nil,
				Position:   nil,
			},
		},
		Type: &ast.Type{
			NamedType: getPayloadObject(s, obj).Name,
		},
	}
	filterObj, ok := s.Types[fmt.Sprintf("%sFilterInput", obj.Name)]
	if ok {
		updateDef.Arguments = append(updateDef.Arguments, &ast.ArgumentDefinition{
			Description: "Filter objects to update",
			Name:        "filter",
			Type:        &ast.Type{NamedType: filterObj.Name},
		})
	}

	return updateDef
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
