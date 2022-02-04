package augmenters

import (
	"fmt"
	"log"
	"strings"

	"github.com/jinzhu/inflection"

	"github.com/roneli/fastgql/gql"
	"github.com/spf13/cast"
	"github.com/vektah/gqlparser/v2/ast"
)

type Aggregation struct {
}

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
		if gql.IsScalarListType(s, f.Type) {
			continue
		}
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

		fieldObj := s.Types[gql.GetType(f.Type).Name()]
		aggregateDef := a.buildAggregateObject(s, fieldObj)

		obj.Fields = append(obj.Fields, &ast.FieldDefinition{
			Name:        aggregateName,
			Description: fmt.Sprintf("%s Aggregate", f.Name),
			Type: &ast.Type{
				NamedType: aggregateDef.Name,
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

func (a Aggregation) buildAggregateObject(s *ast.Schema, obj *ast.Definition) *ast.Definition {
	payloadObjectName := fmt.Sprintf("%sAggregate", inflection.Plural(obj.Name))
	if payloadObject, ok := s.Types[payloadObjectName]; ok {
		return payloadObject
	}
	payloadObject := &ast.Definition{
		Kind:        ast.Object,
		Description: fmt.Sprintf("Aggregate %s", obj.Name),
		Name:        payloadObjectName,
		Fields: []*ast.FieldDefinition{
			{
				Description: "Count results",
				Name:        "count",
				Type: &ast.Type{
					NamedType: "Int",
					NonNull:   true,
				},
			},
		},
	}
	// Add other aggregate functions max/min
	payloadObject.Fields = append(payloadObject.Fields, buildMinMaxField(s, obj)...)

	s.Types[payloadObjectName] = payloadObject
	return payloadObject
}

func buildMinMaxField(s *ast.Schema, obj *ast.Definition) []*ast.FieldDefinition {
	log.Printf("building min/max aggregates for %s\n", obj.Name)
	minObjName := fmt.Sprintf("%sMin", obj.Name)
	minObj := &ast.Definition{
		Kind:        ast.Object,
		Description: fmt.Sprintf("min aggregator for %s", obj.Name),
		Name:        minObjName,
		Fields:      nil,
	}

	maxObjName := fmt.Sprintf("%sMin", obj.Name)
	maxObj := &ast.Definition{
		Kind:        ast.Object,
		Description: fmt.Sprintf("max aggregator for %s", obj.Name),
		Name:        maxObjName,
		Fields:      nil,
	}

	for _, f := range obj.Fields {
		if gql.IsListType(f.Type) {
			continue
		}
		t := gql.GetType(f.Type)
		fieldDef := s.Types[t.Name()]
		// we only support scalar types as aggregate fields
		if !fieldDef.IsLeafType() {
			continue
		}
		log.Printf("adding field %s to min/max aggregates for %s\n", f.Name, obj.Name)
		minObj.Fields = append(minObj.Fields, &ast.FieldDefinition{
			Description: fmt.Sprintf("Compute the minimum for %s", f.Name),
			Name:        f.Name,
			Type: &ast.Type{
				NamedType: t.NamedType,
				NonNull:   true,
			},
		})
		maxObj.Fields = append(maxObj.Fields, &ast.FieldDefinition{
			Description: fmt.Sprintf("Compute the maxiumum for %s", f.Name),
			Name:        f.Name,
			Type: &ast.Type{
				NamedType: t.NamedType,
				NonNull:   true,
			},
		})
	}
	// add object to schema
	s.Types[minObjName] = minObj
	s.Types[maxObjName] = maxObj
	return []*ast.FieldDefinition{
		{
			Description: "Computes the maximum of the non-null input values.",
			Name:        "max",
			Type: &ast.Type{
				NamedType: maxObjName,
			},
		},
		{
			Description: "Computes the minimum of the non-null input values.",
			Name:        "min",
			Type: &ast.Type{
				NamedType: minObjName,
			},
		},
	}
}
