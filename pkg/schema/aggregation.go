package schema

import (
	"fmt"
	"github.com/iancoleman/strcase"
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
	aggDef := addAggregateObject(s, fieldDef)
	if aggDef == nil {
		log.Printf("aggreationField for field %s@%s already exists skipping\n", field.Name, obj.Name)
	}
	// add the type to the schema
	s.Types[aggDef.Name] = aggDef
	aggregateName := fmt.Sprintf("_%sAggregate", field.Name)
	// check if field already exists, if so, skip
	if def := obj.Fields.ForName(aggregateName); def != nil {
		log.Printf("aggreationField for field %s@%s already exists skipping\n", field.Name, obj.Name)
		if def.Directives.ForName("generate") == nil {
			// add directive to field, so filter can be generated
			def.Directives = append(def.Directives, addGenerateDirective(s))
		}
		return
	}
	addAggregateField(s, obj, field, aggDef)
	log.Printf("adding aggregation field to field %s@%s\n", field.Name, obj.Name)

}

func addAggregateField(s *ast.Schema, obj *ast.Definition, field *ast.FieldDefinition, aggDef *ast.Definition) {
	aggregateName := fmt.Sprintf("_%sAggregate", field.Name)
	obj.Fields = append(obj.Fields, &ast.FieldDefinition{
		Name:        aggregateName,
		Description: fmt.Sprintf("%s Aggregate", field.Name),
		Arguments: ast.ArgumentDefinitionList{
			{
				Name: "groupBy",
				Type: &ast.Type{
					Elem: &ast.Type{
						NamedType: fmt.Sprintf("%sGroupBy", strcase.ToCamel(field.Type.Name())),
						NonNull:   true,
					},
				},
			},
		},
		Type: &ast.Type{
			Elem: &ast.Type{
				NamedType: aggDef.Name,
				NonNull:   true,
			},
			NonNull: true,
		},
		// add directive to field, so filter can be generated
		Directives: ast.DirectiveList{addGenerateDirective(s)},
	})
}

func addGenerateDirective(s *ast.Schema) *ast.Directive {
	return &ast.Directive{
		Name: "generate",
		Arguments: []*ast.Argument{
			{
				Name: "filter",
				Value: &ast.Value{
					Raw:  "true",
					Kind: ast.BooleanValue,
				},
			},
		},
		Definition: s.Directives["generate"],
	}
}

// addAggregateGroupByObject builds the group by object for the aggregate
func addAggregateGroupByObject(s *ast.Schema, obj *ast.Definition) {
	// check if group by object already exists, if so, skip
	if _, ok := s.Types[fmt.Sprintf("%sGroupBy", obj.Name)]; ok {
		log.Printf("group by object for %s already exists skipping\n", obj.Name)
		return
	}
	groupBy := &ast.Definition{
		Kind:        ast.Enum,
		Description: fmt.Sprintf("Group by %s", obj.Name),
		Name:        fmt.Sprintf("%sGroupBy", obj.Name),
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
		log.Printf("adding field %s to group by aggregates for %s\n", f.Name, obj.Name)
		groupBy.EnumValues = append(groupBy.EnumValues, &ast.EnumValueDefinition{
			Description: fmt.Sprintf("Group by %s", f.Name),
			Name:        strcase.ToScreamingSnake(f.Name),
		})
	}
	// add object to schema
	s.Types[groupBy.Name] = groupBy
	addRecursiveAggregation(s, obj)
}

func addRecursiveAggregation(s *ast.Schema, obj *ast.Definition) {
	for _, f := range obj.Fields {
		// aggregate only on fields with the @relation directive
		if f.Directives.ForName("relation") == nil {
			continue
		}
		def := s.Types[f.Type.Name()]
		aggDef := addAggregateObject(s, def)
		if def != nil {
			addAggregateField(s, obj, f, aggDef)
		}
	}
}

func addAggregateObject(s *ast.Schema, obj *ast.Definition) *ast.Definition {
	payloadObjectName := fmt.Sprintf("%sAggregate", inflection.Plural(obj.Name))
	// Add group by if not exists
	addAggregateGroupByObject(s, obj)
	if payloadObject, ok := s.Types[payloadObjectName]; ok {
		return payloadObject
	}
	payloadObject := &ast.Definition{
		Kind:        ast.Object,
		Description: fmt.Sprintf("Aggregate %s", obj.Name),
		Name:        payloadObjectName,
		Fields: []*ast.FieldDefinition{
			{
				Description: "Group",
				Name:        "group",
				Type: &ast.Type{
					NamedType: "Map",
					NonNull:   false,
				},
			},
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
```