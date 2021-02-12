package builders

import (
	"github.com/roneli/fastgql/internal"
	"github.com/roneli/fastgql/schema"
	"github.com/vektah/gqlparser/v2/ast"
)

type (

	// Config is the basic level of data passed to a builder when it's created
	Config struct {
		Schema   *ast.Schema
		Logger   internal.Logger
		LogLevel internal.LogLevel
	}

	Builder interface {
		Config() *Config
	}

	// FieldBuilder allows to call collect field to go over fields requested in a query
	FieldBuilder interface {
		Builder
		OnSingleField(f *ast.Field, variables map[string]interface{}) error
		OnSelectionField(f *ast.Field, variables map[string]interface{}) error
	}

	// PaginationBuilder allows to add pagination based on pagination arguments passed in a query
	PaginationBuilder interface {
		Builder
		Limit(limit uint) error
		Offset(offset uint) error
	}

	OrderingTypes string

	OrderField struct {
		Key  string
		Type OrderingTypes
	}

	OrderingBuilder interface {
		OrderBy([]OrderField) error
	}

	// FilterBuilder allow builders to support condition building
	FilterBuilder interface {
		Builder
		// Operation is called when a simple operator is found
		Operation(name, op string, value interface{}) error
		// Filter is called when the operation is a BoolExp
		Filter(f *ast.FieldDefinition, key string, value map[string]interface{}) error
		// Logical is called when the operation is AND, OR, NOT
		Logical(f *ast.FieldDefinition, logicalExp schema.LogicalOperator, values []interface{}) error
	}

	// ArgumentsBuilder allows Builders to build arguments supported by FastGQL
	ArgumentsBuilder interface {
		PaginationBuilder
		FilterBuilder
		OrderingBuilder
	}

	// ArgumentsBuilder allows Builders to build aggregate queries on _XYZAggregate fields
	AggregateBuilder interface {
		FieldBuilder
		FilterBuilder
		Aggregate(f *ast.Field) error
	}

	// QueryBuilder supports building a full query from a given GraphQL query
	QueryBuilder interface {
		ArgumentsBuilder
		FieldBuilder
		Query() (string, []interface{}, error)
	}
)

const (
	OrderingTypesAsc      OrderingTypes = "ASC"
	OrderingTypesDesc     OrderingTypes = "DESC"
	OrderingTypesAscNull  OrderingTypes = "ASC_NULL_FIRST"
	OrderingTypesDescNull OrderingTypes = "DESC_NULL_FIRST"
)
