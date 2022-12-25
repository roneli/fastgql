package mongo

import (
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
)

type Operator func(field, operator string, value interface{}) bson.E

var defaultOperators = map[string]Operator{
	"eq":     commonOperator,
	"neq":    operator("ne"),
	"like":   Like,
	"ilike":  ILike,
	"notIn":  NotIn,
	"in":     In,
	"isNull": IsNull,
	"gt":     commonOperator,
	"gte":    commonOperator,
	"lte":    commonOperator,
	"lt":     commonOperator,
}

func commonOperator(field, operator string, value interface{}) bson.E {
	return bson.E{Key: field, Value: bson.D{{Key: fmt.Sprintf("$%s", operator), Value: value}}}
}
func operator(operator string) Operator {
	return func(field, _ string, value interface{}) bson.E {
		return bson.E{Key: field, Value: bson.D{{Key: fmt.Sprintf("$%s", operator), Value: value}}}
	}
}

func Like(field, _ string, value interface{}) bson.E {
	return bson.E{Key: field, Value: bson.D{{Key: "$regex", Value: value}}}
}

func ILike(field, _ string, value interface{}) bson.E {
	return bson.E{Key: field, Value: bson.D{{Key: "$regex", Value: value}, {Key: "$options", Value: "i"}}}
}

func In(field, _ string, value interface{}) bson.E {
	return bson.E{Key: field, Value: bson.D{{Key: "$in", Value: value}}}
}

func NotIn(field, _ string, value interface{}) bson.E {
	return bson.E{Key: field, Value: bson.D{{Key: "$nin", Value: value}}}
}

func IsNull(field, _ string, value interface{}) bson.E {
	return bson.E{Key: field, Value: bson.D{{Key: "$exists", Value: value}}}
}
