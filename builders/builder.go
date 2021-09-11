package builders

import (
	"math/rand"
	"unsafe"

	"github.com/roneli/fastgql/log"

	"github.com/vektah/gqlparser/v2/ast"
)

type (

	// Config is the basic level of data passed to a builder when it's created
	Config struct {
		Schema             *ast.Schema
		Logger             log.Logger
		TableNameGenerator TableNameGenerator
	}

	OrderingTypes string

	OrderField struct {
		Key  string
		Type OrderingTypes
	}

	// AggregateBuilder allows Builders to build aggregate queries on _XYZAggregate fields
	AggregateBuilder interface {
		Aggregate(field Field) (string, []interface{}, error)
	}

	// QueryBuilder supports building a full query from a given GraphQL query
	QueryBuilder interface {
		Query(field Field) (string, []interface{}, error)
	}

	TableNameGenerator interface {
		Generate(n int) string
	}
)

const (
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
