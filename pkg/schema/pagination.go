package schema

import (
	"github.com/spf13/cast"
	"github.com/vektah/gqlparser/v2/ast"
	"log"
)

func PaginationAugmenter(s *ast.Schema) error {
	for _, v := range s.Query.Fields {
		d := v.Directives.ForName("generate")
		if d == nil {
			continue
		}
		if !IsListType(v.Type) {
			continue
		}
		log.Printf("adding pagination to field %s@%s\n", v.Name, s.Query.Name)
		args := d.ArgumentMap(nil)
		if p, ok := args["pagination"]; ok && cast.ToBool(p) {
			addPaginationToField(s.Query, v)
		}
		if recursive := cast.ToBool(args["recursive"]); recursive {
			addPagination(s, s.Types[GetType(v.Type).Name()], s.Query, recursive)
		}
	}
	return nil
}

func addPaginationToField(obj *ast.Definition, field *ast.FieldDefinition) {
	if skipAugment(field, "limit", "offset") {
		return
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
}

func addPagination(s *ast.Schema, obj *ast.Definition, parent *ast.Definition, recursive bool) {
	for _, f := range obj.Fields {
		// avoid recurse
		if skipAugment(f, "limit", "offset") {
			continue
		}
		fieldType := s.Types[f.Type.Name()]
		if IsScalarListType(s, f.Type) || !IsListType(f.Type) {
			if recursive && fieldType.IsCompositeType() && fieldType != parent {
				addPagination(s, fieldType, obj, recursive)
			}
			continue
		}
		log.Printf("adding pagination to field %s@%s\n", f.Name, obj.Name)
		addPaginationToField(obj, f)
		if recursive && fieldType.IsCompositeType() {
			log.Printf("adding recursive pagination to field %s@%s\n", f.Name, obj.Name)
			addPagination(s, fieldType, obj, recursive)
		}
	}
}
