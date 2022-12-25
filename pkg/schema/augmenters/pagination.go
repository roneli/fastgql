package augmenters

import (
	"log"
	"strings"

	"github.com/roneli/fastgql/pkg/schema/gql"

	"github.com/spf13/cast"
	"github.com/vektah/gqlparser/v2/ast"
)

// Pagination Augmenter allows adding pagination (limit/offset) augmenting any objects' field with @generate if it's a List
type Pagination struct{}

func (p Pagination) DirectiveName() string {
	return "generate"
}

func (p Pagination) Augment(s *ast.Schema) error {
	for _, v := range s.Types {
		d := v.Directives.ForName(p.DirectiveName())
		if d == nil {
			continue
		}

		args := d.ArgumentMap(nil)
		recursive := cast.ToBool(args["recursive"])
		if addPagination, ok := args["pagination"]; ok && cast.ToBool(addPagination) {
			p.addPagination(s, v, nil, recursive)
		}
	}
	return nil
}

func (p Pagination) addPagination(s *ast.Schema, obj *ast.Definition, parent *ast.Definition, recursive bool) {
	for _, f := range obj.Fields {
		// avoid recurse
		if strings.HasPrefix(f.Name, "__") || f.Arguments.ForName("limit") != nil || f.Arguments.ForName("offset") != nil {
			continue
		}

		fieldType := s.Types[f.Type.Name()]
		if gql.IsScalarListType(s, f.Type) || !gql.IsListType(f.Type) {
			if recursive && fieldType.IsCompositeType() && fieldType != parent {
				p.addPagination(s, fieldType, obj, recursive)
			}
			continue
		}
		log.Printf("adding pagination to field %s@%s\n", f.Name, obj.Name)
		f.Arguments = append(f.Arguments,
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
		if recursive && fieldType.IsCompositeType() {
			log.Printf("adding recursive pagination to field %s@%s\n", f.Name, obj.Name)
			p.addPagination(s, fieldType, obj, recursive)
		}
	}
}
