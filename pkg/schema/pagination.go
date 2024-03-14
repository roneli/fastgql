package schema

import (
	"log"

	"github.com/spf13/cast"
	"github.com/vektah/gqlparser/v2/ast"
)

func PaginationAugmenter(s *ast.Schema) error {
	for _, v := range s.Query.Fields {
		d := v.Directives.ForName(generateDirectiveName)
		if d == nil {
			continue
		}
		if !IsListType(v.Type) {
			continue
		}
		log.Printf("adding pagination to field %s@%s\n", v.Name, s.Query.Name)
		args := d.ArgumentMap(nil)
		if p, ok := args["pagination"]; ok && cast.ToBool(p) {
			if err := addPaginationToField(s, s.Query, v); err != nil {
				return err
			}
		}
		if recursive := cast.ToBool(args["recursive"]); recursive {
			if err := addRecursive(s, s.Types[GetType(v.Type).Name()], "limit", addPaginationToField); err != nil {
				return err
			}
		}
	}
	return nil
}

func addPaginationToField(_ *ast.Schema, obj *ast.Definition, field *ast.FieldDefinition) error {
	if skipAugment(field, "limit", "offset") {
		return nil
	}
	log.Printf("adding pagination to field %s@%s\n", field.Name, obj.Name)
	field.Arguments = append(field.Arguments,
		&ast.ArgumentDefinition{Description: "Limit",
			Name:         "limit",
			DefaultValue: &ast.Value{Raw: "100", Kind: ast.IntValue},
			Type:         &ast.Type{NamedType: "Int"},
		},
		&ast.ArgumentDefinition{
			Description:  "Offset",
			Name:         "offset",
			DefaultValue: &ast.Value{Raw: "0", Kind: ast.IntValue},
			Type:         &ast.Type{NamedType: "Int"},
		},
	)
	return nil
}
