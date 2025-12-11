package schema

import (
	"fmt"

	"github.com/spf13/cast"
	"github.com/vektah/gqlparser/v2/ast"
)

type ArgName string

const (
	GroupBy     ArgName = "groupBy"
	FilterInput ArgName = "filter"
	OrderBy     ArgName = "orderBy"
)

// Directive names used in fastgql GraphQL schemas
const (
	GenerateDirectiveName            = "generate"
	SkipGenerateDirectiveName        = "skipGenerate"
	TableDirectiveName               = "table"
	RelationDirectiveName            = "relation"
	JSONDirectiveName                = "json"
	GenerateMutationsDirectiveName   = "generateMutations"
	GenerateFilterInputDirectiveName = "generateFilterInput"
	IsInterfaceFilterDirectiveName   = "isInterfaceFilter"
	FastGQLFieldDirectiveName        = "fastgqlField"
)

// Directive argument names - exported for external packages
const (
	ArgNameTable            = "name"
	ArgNameDialect          = "dialect"
	ArgNameSchema           = "schema"
	ArgNameType             = "type"
	ArgNameFields           = "fields"
	ArgNameReferences       = "references"
	ArgNameColumn           = "column"
	ArgNameBaseTable        = "baseTable"
	ArgNameRefTable         = "refTable"
	ArgNameManyToManyTable  = "manyToManyTable"
	ArgNameManyToManyFields = "manyToManyFields"
	ArgNameManyToManyRefs   = "manyToManyReferences"
)

// GraphQL type names - exported for external packages
const (
	GraphQLTypeQuery   = "Query"
	GraphQLTypeInt     = "Int"
	GraphQLTypeString  = "String"
	GraphQLTypeBoolean = "Boolean"
	GraphQLTypeFloat   = "Float"
	GraphQLTypeID      = "ID"
	GraphQLTypeMap     = "Map" // Custom scalar for fastgql
)

type TableDirective struct {
	// Name of the table/collection
	Name string
	// Schema name table resides in, can be omitted
	Schema string
	// Dialect name the table resides in
	Dialect string
}

type RelationType string

const (
	OneToMany  RelationType = "ONE_TO_MANY"
	OneToOne   RelationType = "ONE_TO_ONE"
	ManyToMany RelationType = "MANY_TO_MANY"
)

type RelationDirective struct {
	RelType              RelationType
	BaseTable            string
	ReferenceTable       string
	Fields               []string
	References           []string
	ManyToManyTable      string
	ManyToManyReferences []string
	ManyToManyFields     []string
}

type JSONDirective struct {
	Column string
}

func GetTableDirective(def *ast.Definition) (*TableDirective, error) {
	d := def.Directives.ForName(TableDirectiveName)
	if d == nil {
		return nil, fmt.Errorf("failed to get table directive for %s", def.Name)
	}
	return &TableDirective{
		Name:    getArgumentValue(d.Arguments, ArgNameTable),
		Schema:  getArgumentValue(d.Arguments, ArgNameSchema),
		Dialect: getArgumentValue(d.Arguments, ArgNameDialect),
	}, nil
}

func GetRelationDirective(field *ast.FieldDefinition) *RelationDirective {
	d := field.Directives.ForName(RelationDirectiveName)
	if d == nil {
		return nil
	}
	relType := d.Arguments.ForName(ArgNameType).Value.Raw
	return &RelationDirective{
		RelType:              RelationType(relType),
		Fields:               cast.ToStringSlice(GetDirectiveValue(d, ArgNameFields)),
		References:           cast.ToStringSlice(GetDirectiveValue(d, ArgNameReferences)),
		BaseTable:            cast.ToString(GetDirectiveValue(d, ArgNameBaseTable)),
		ReferenceTable:       cast.ToString(GetDirectiveValue(d, ArgNameRefTable)),
		ManyToManyTable:      cast.ToString(GetDirectiveValue(d, ArgNameManyToManyTable)),
		ManyToManyFields:     cast.ToStringSlice(GetDirectiveValue(d, ArgNameManyToManyFields)),
		ManyToManyReferences: cast.ToStringSlice(GetDirectiveValue(d, ArgNameManyToManyRefs)),
	}
}

func GetJSONDirective(field *ast.FieldDefinition) *JSONDirective {
	d := field.Directives.ForName(JSONDirectiveName)
	if d == nil {
		return nil
	}
	return &JSONDirective{
		Column: cast.ToString(GetDirectiveValue(d, ArgNameColumn)),
	}
}

func getArgumentValue(args ast.ArgumentList, name string) string {
	arg := args.ForName(name)
	if arg == nil {
		return ""
	}
	return arg.Value.Raw
}
