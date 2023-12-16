package schema

import (
	"fmt"
	"log"

	"github.com/jinzhu/inflection"
	"github.com/spf13/cast"
	"github.com/vektah/gqlparser/v2/ast"
)

type Aggregation struct{}

func (a Aggregation) DirectiveName() string {
	return "generate"
}

func (a Aggregation) Name() string {
	return "aggregation"
}

func AggregationAugmenter(s *ast.Schema) error {
	for _, v := range s.Query.Fields {
		d := v.Directives.ForName("generate")
		if d == nil {
			continue
		}
		if !IsListType(v.Type) {
			continue
		}
		log.Printf("adding aggregation field to query %s@%s\n", v.Name, s.Query.Name)
		args := d.ArgumentMap(nil)
		if p, ok := args["aggregate"]; ok && cast.ToBool(p) {
			addAggregationField(s, s.Query, v)
		}
		// TODO: add recursive aggregation
	}
	return nil
}

func addAggregationField(s *ast.Schema, obj *ast.Definition, field *ast.FieldDefinition) {
	t := GetType(field.Type)
	fieldDef, ok := s.Types[t.Name()]
	if !ok || !fieldDef.IsCompositeType() {
		return
	}
	aggDef := buildAggregateObject(s, fieldDef)
	if aggDef == nil {
		log.Printf("aggreationField for field %s@%s already exists skipping\n", field.Name, obj.Name)
	}
	// add the type to the schema
	s.Types[aggDef.Name] = aggDef
	aggregateName := fmt.Sprintf("_%sAggregate", field.Name)
	// check if field already exists, if so, skip
	if def := obj.Fields.ForName(aggregateName); def != nil {
		log.Printf("aggreationField for field %s@%s already exists skipping\n", field.Name, obj.Name)
		return
	}
	log.Printf("adding aggregation field to field %s@%s\n", field.Name, obj.Name)
	obj.Fields = append(obj.Fields, &ast.FieldDefinition{
		Name:        aggregateName,
		Description: fmt.Sprintf("%s Aggregate", field.Name),
		Type: &ast.Type{
			NamedType: aggDef.Name,
			NonNull:   true,
		},
	})
}

func buildAggregateObject(s *ast.Schema, obj *ast.Definition) *ast.Definition {
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
		if IsListType(f.Type) {
			continue
		}
		t := GetType(f.Type)
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