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

const (
	generateDirectiveName     = "generate"
	skipGenerateDirectiveName = "skipGenerate"
	tableDirectiveName        = "table"
	relationDirectiveName     = "relation"
	jsonDirectiveName         = "json"
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

func GetTableDirective(def *ast.Definition) (*TableDirective, error) {
	d := def.Directives.ForName("table")
	if d == nil {
		return nil, fmt.Errorf("failed to get table directive for %s", def.Name)
	}
	return &TableDirective{
		Name:    getArgumentValue(d.Arguments, "name"),
		Schema:  getArgumentValue(d.Arguments, "schema"),
		Dialect: getArgumentValue(d.Arguments, "dialect"),
	}, nil
}

func GetRelationDirective(field *ast.FieldDefinition) *RelationDirective {
	d := field.Directives.ForName(relationDirectiveName)
	if d == nil {
		return nil
	}
	relType := d.Arguments.ForName("type").Value.Raw
	return &RelationDirective{
		RelType:              RelationType(relType),
		Fields:               cast.ToStringSlice(GetDirectiveValue(d, "fields")),
		References:           cast.ToStringSlice(GetDirectiveValue(d, "references")),
		BaseTable:            cast.ToString(GetDirectiveValue(d, "baseTable")),
		ReferenceTable:       cast.ToString(GetDirectiveValue(d, "refTable")),
		ManyToManyTable:      cast.ToString(GetDirectiveValue(d, "manyToManyTable")),
		ManyToManyFields:     cast.ToStringSlice(GetDirectiveValue(d, "manyToManyFields")),
		ManyToManyReferences: cast.ToStringSlice(GetDirectiveValue(d, "manyToManyReferences")),
	}
}

func getArgumentValue(args ast.ArgumentList, name string) string {
	arg := args.ForName(name)
	if arg == nil {
		return ""
	}
	return arg.Value.Raw
}
