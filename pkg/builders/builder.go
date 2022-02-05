package builders

import (
	"math/rand"
	"unsafe"

	"github.com/roneli/fastgql/internal/log"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"

	"github.com/vektah/gqlparser/v2/ast"
)

type (
	// Config is the basic level of data passed to a builder when it's created
	Config struct {
		Schema             *ast.Schema
		Logger             log.Logger
		TableNameGenerator TableNameGenerator
		// CustomOperators are user defined operators, can also be used to override existing default operators.
		CustomOperators map[string]Operator
	}

	OrderingTypes string

	OrderField struct {
		Key  string
		Type OrderingTypes
	}

	// Operator gets called on filters expressions written in graphql. Users can define new operators in the graphql
	// schema, and define operator functions for those operators based on the operator name given.
	// Operators are added by "key" to comparator Input types, and get called with expected value.
	Operator func(table exp.AliasedExpression, key string, value interface{}) goqu.Expression

	// AggregatorOperator gets called on aggregation methods // TBD //
	AggregatorOperator func(table exp.AliasedExpression, fields []Field) (goqu.Expression, error)

	// AggregateBuilder allows Builders to build aggregate queries on _XYZAggregate fields
	AggregateBuilder interface {
		Aggregate(field Field) (string, []interface{}, error)
	}

	// QueryBuilder supports building a full query from a given GraphQL query
	QueryBuilder interface {
		Query(field Field) (string, []interface{}, error)
	}

	// MutationBuilder supports building DELETE/CREATE/UPDATE queries from given GraphQL
	MutationBuilder interface {
		Create(field Field) (string, []interface{}, error)
		Delete(field Field) (string, []interface{}, error)
	}

	TableNameGenerator interface {
		Generate(n int) string
	}
)

const (
	InputFieldName = "inputs"

	OrderingTypesAsc      OrderingTypes = "ASC"
	OrderingTypesDesc     OrderingTypes = "DESC"
	OrderingTypesAscNull  OrderingTypes = "ASC_NULL_FIRST"
	OrderingTypesDescNull OrderingTypes = "DESC_NULL_FIRST"

	// GenerateTableName configuration
	letterBytes   = "abcdefghijklmnopqrstuvwxyz"
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

func GenerateTableName(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, rand.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = rand.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return *(*string)(unsafe.Pointer(&b))
}
