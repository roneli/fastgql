package augmenters

import (
	"github.com/roneli/fastgql/gql"
	"github.com/spf13/cast"
	"github.com/vektah/gqlparser/v2/ast"
	"strings"
)

type Pagination struct {}

func (p Pagination) Name() string {
	return "generateArguments"
}

func (p Pagination) Augment(s *ast.Schema) error {
	for _, v := range s.Types {
		d := v.Directives.ForName(p.Name())
		if d == nil {
			continue
		}

		args := d.ArgumentMap(nil)
		recursive := cast.ToBool(args["recursive"])
		if addPagination, ok := args["pagination"]; ok && cast.ToBool(addPagination) {
			p.addPagination(s, v, recursive)
		}
	}
	return nil
}

func (p Pagination) addPagination(s *ast.Schema, obj *ast.Definition, recursive bool) {
	for _, f := range obj.Fields {
		if strings.HasPrefix(f.Name, "__") {
			continue
		}
		if !gql.IsListType(f.Type) {
			continue
		}

		if f.Arguments.ForName("limit") != nil || f.Arguments.ForName("offset") != nil {
			continue
		}

		f.Arguments = append(f.Arguments,
			&ast.ArgumentDefinition{Description: "Limit",
				Name:         "limit",
				DefaultValue: &ast.Value{Raw: "100", Kind: ast.IntValue},
				Type:         &ast.Type{NamedType: "Int"},
			},
			&ast.ArgumentDefinition{
				Description: "Offset",
				Name: "offset",
				DefaultValue: &ast.Value{Raw: "0", Kind: ast.IntValue},
				Type: &ast.Type{NamedType: "Int"},
			},
		)
		if !recursive {
			continue
		}
		fieldType := s.Types[f.Type.Name()]
		if !fieldType.IsCompositeType() {
			continue
		}
		p.addPagination(s, fieldType, recursive)
	}
}
