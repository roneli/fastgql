package augmenters

import (
	"fmt"
	"strings"

	"github.com/roneli/fastgql/gql"
	"github.com/spf13/cast"
	"github.com/vektah/gqlparser/v2/ast"
)

type Aggregation struct{}

func (a Aggregation) DirectiveName() string {
	return "generate"
}

func (a Aggregation) Augment(s *ast.Schema) error {
	for _, v := range s.Types {
		d := v.Directives.ForName(a.DirectiveName())
		if d == nil {
			continue
		}

		args := d.ArgumentMap(nil)
		recursive := cast.ToBool(args["recursive"])
		if addAggregate, ok := args["aggregate"]; ok && cast.ToBool(addAggregate) {
			a.addAggregation(s, v, recursive)
		}
	}
	return nil
}

func (a Aggregation) addAggregation(s *ast.Schema, obj *ast.Definition, recursive bool) {

	for _, f := range obj.Fields {
		if strings.HasPrefix(f.Name, "__") {
			continue
		}
		if !gql.IsListType(f.Type) {
			continue
		}

		aggregateName := fmt.Sprintf("_%sAggregate", f.Name)
		if def := obj.Fields.ForName(aggregateName); def != nil {
			continue
		}

		obj.Fields = append(obj.Fields, &ast.FieldDefinition{
			Name:        aggregateName,
			Description: fmt.Sprintf("%s Aggregate", f.Name),
			Type: &ast.Type{
				NamedType: "_AggregateResult",
				NonNull:   true,
			},
		})

		if !recursive {
			continue
		}
		fieldType := s.Types[f.Type.Name()]
		if !fieldType.IsCompositeType() {
			continue
		}
		a.addAggregation(s, fieldType, recursive)
	}
}
