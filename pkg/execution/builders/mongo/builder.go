package mongo

import (
	"fmt"

	"github.com/roneli/fastgql/pkg/log"
	"github.com/spf13/cast"

	"go.mongodb.org/mongo-driver/mongo"

	"github.com/roneli/fastgql/pkg/execution/builders"
	"github.com/vektah/gqlparser/v2/ast"
	"go.mongodb.org/mongo-driver/bson"
)

type Builder struct {
	Schema              *ast.Schema
	Logger              log.Logger
	TableNameGenerator  builders.TableNameGenerator
	Operators           map[string]Operator
	AggregatorOperators map[string]builders.AggregatorOperator
	CaseConverter       builders.ColumnCaseConverter
}

func NewBuilder(config *builders.Config) Builder {
	var l log.Logger = log.NullLogger{}
	if config.Logger != nil {
		l = config.Logger
	}
	operators := make(map[string]Operator)
	for k, v := range defaultOperators {
		operators[k] = v
	}
	return Builder{Schema: config.Schema, Operators: operators, Logger: l}
}

// TODO: issues:
// 1. first argument is string not used
// 2. second argument is []interface i.e args also not used and changed to be something specific.
// 3. operators is not generic enough

func (b Builder) Query(field builders.Field) (mongo.Pipeline, error) {
	pipeline := mongo.Pipeline{}
	filters, err := b.buildFilter(field)
	if err != nil {
		return nil, fmt.Errorf("failed to build filters: %w", err)
	}
	if len(filters) > 0 {
		pipeline = append(pipeline, filters)
	}

	pagination := b.buildPagination(field)
	if len(pagination) > 0 {
		pipeline = append(pipeline, pagination...)
	}

	projection, err := b.buildProjection(field)
	if err != nil {
		return nil, fmt.Errorf("failed to build filters: %w", err)
	}
	pipeline = append(pipeline, projection)

	return pipeline, nil
}

func (b Builder) buildPagination(field builders.Field) []bson.D {
	var pagination []bson.D
	if offset, ok := field.Arguments["offset"]; ok {
		b.Logger.Debug("adding pagination offset", "offset", offset)
		pagination = append(pagination, bson.D{{Key: "$skip", Value: cast.ToUint(offset)}})
	}
	if limit, ok := field.Arguments["limit"]; ok {
		b.Logger.Debug("adding pagination limit", "limit", limit)
		pagination = append(pagination, bson.D{{Key: "$limit", Value: cast.ToUint(limit)}})
	}
	return pagination
}

func (b Builder) buildFilter(field builders.Field) (bson.D, error) {
	filterArg, ok := field.Arguments["filter"]
	if !ok {
		return bson.D{}, nil
	}
	filters, ok := filterArg.(map[string]interface{})
	if !ok {
		return bson.D{}, fmt.Errorf("unexpected filter arg type")
	}
	f, err := b.buildFilterExp(field, filters)
	if err != nil {
		return bson.D{}, err
	}
	return bson.D{{Key: "$match", Value: f}}, nil
}

func (b Builder) buildFilterExp(_ builders.Field, filters map[string]interface{}) (bson.D, error) {
	var allFilters bson.D
	for k, v := range filters {
		switch {
		case k == string(builders.LogicalOperatorAND) || k == string(builders.LogicalOperatorOR):
			panic("not implemented")
		case k == string(builders.LogicalOperatorNot):
			panic("not implemented")
		default:
			opMap, ok := v.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("fatal value of key not map")
			}
			for op, value := range opMap {
				opExp, err := b.Operation(k, op, value)
				if err != nil {
					return nil, err
				}
				allFilters = append(allFilters, opExp)
			}
		}
	}
	return allFilters, nil
}

func (b Builder) Operation(fieldName, operatorName string, value interface{}) (bson.E, error) {
	opFunc, ok := b.Operators[operatorName]
	if !ok {
		return bson.E{}, fmt.Errorf("key operator %s not supported", operatorName)
	}
	return opFunc(fieldName, operatorName, value), nil
}

func (b Builder) buildProjection(field builders.Field) (bson.D, error) {
	p, err := b.doBuildProjection(field)
	if err != nil {
		return nil, fmt.Errorf("failed to build projection: %w", err)
	}
	return bson.D{{Key: "$project", Value: p}}, nil
}

func (b Builder) doBuildProjection(field builders.Field) (bson.M, error) {
	var projection = bson.M{}
	for _, childField := range field.Selections {
		switch childField.FieldType {
		case builders.TypeScalar:
			b.Logger.Debug("adding field", "collection", "", "fieldName", childField.Name)
			projection[childField.Name] = 1
		case builders.TypeRelation:
			b.Logger.Debug("adding relation field", "fieldName", childField.Name)

			subProjection, err := b.doBuildProjection(childField)
			if err != nil {
				return nil, fmt.Errorf("failed to build relation for %s", childField.Name)
			}
			for k, v := range subProjection {
				projection[childField.Name+"."+k] = v
			}
		case builders.TypeAggregate:
			aggP, err := b.buildProjectionAggregate(field, childField)
			if err != nil {
				return nil, fmt.Errorf("failed to build relation for %s", childField.Name)
			}
			for k, v := range aggP {
				projection[k] = v
			}
		default:
			b.Logger.Error("unknown field type", "fieldName", childField.Name, "fieldType", childField.FieldType)
			panic("unknown field type")
		}
	}
	return projection, nil
}

func (b Builder) buildProjectionAggregate(parentField, field builders.Field) (bson.M, error) {
	var ap = make(bson.M, len(field.Selections))
	naf := builders.GetAggregateField(parentField, field)
	f := fmt.Sprintf("$%s", naf.Name)
	for _, s := range field.Selections {
		switch s.Name {
		//   count: {"$size": {"$cond": {if: {"$isArray": "$films"}, then: "$films", else: []}}}
		case "count":
			ap[field.Name+".count"] = bson.D{{Key: "$size", Value: bson.M{"$cond": bson.M{"if": bson.D{{Key: "$isArray", Value: f}}, "then": f, "else": bson.A{}}}}}
		default:
			panic("unknown aggregation selection")
		}
	}
	return ap, nil
}
