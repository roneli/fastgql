package builders

import (
	"fmt"

	"github.com/roneli/fastgql/pkg/schema/gql"
	"github.com/spf13/cast"

	"github.com/vektah/gqlparser/v2/ast"
)

type TableDirective struct {
	// Name of the table/collection
	Name string
	// Schema name table resides in, can be omitted
	Schema string
	// Dialect name the table resides in
	Dialect string
}

type DialectDirective struct {
	Dialect       string
	ParentDialect string
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
		Name:    GetArgumentValue(d.Arguments, "name"),
		Schema:  GetArgumentValue(d.Arguments, "schema"),
		Dialect: GetArgumentValue(d.Arguments, "dialect"),
	}, nil
}

func GetRelationDirective(field *ast.FieldDefinition) *RelationDirective {
	d := field.Directives.ForName("relation")
	if d == nil {
		return nil
	}
	relType := d.Arguments.ForName("type").Value.Raw
	return &RelationDirective{
		RelType:              RelationType(relType),
		Fields:               cast.ToStringSlice(gql.GetDirectiveValue(d, "fields")),
		References:           cast.ToStringSlice(gql.GetDirectiveValue(d, "references")),
		BaseTable:            cast.ToString(gql.GetDirectiveValue(d, "baseTable")),
		ReferenceTable:       cast.ToString(gql.GetDirectiveValue(d, "refTable")),
		ManyToManyTable:      cast.ToString(gql.GetDirectiveValue(d, "manyToManyTable")),
		ManyToManyFields:     cast.ToStringSlice(gql.GetDirectiveValue(d, "manyToManyFields")),
		ManyToManyReferences: cast.ToStringSlice(gql.GetDirectiveValue(d, "manyToManyReferences")),
	}
}

func GetDialectDirective(schema *ast.Schema, field Field) (*DialectDirective, error) {
	typeName := field.GetTypeName()
	objType, ok := schema.Types[typeName]
	if !ok {
		return nil, fmt.Errorf("failed to find type definition for %s", typeName)
	}
	parentDialect := "postgres"
	pd := field.ObjectDefinition.Directives.ForName("dialect")
	if pd == nil {
		parentDialect = pd.Arguments.ForName("type").Value.Raw
	}
	d := objType.Directives.ForName("dialect")
	if d == nil {
		return &DialectDirective{
			Dialect:       "postgres",
			ParentDialect: parentDialect,
		}, nil
	}
	return &DialectDirective{
		Dialect:       pd.Arguments.ForName("type").Value.Raw,
		ParentDialect: parentDialect,
	}, nil
}

func GetArgumentValue(args ast.ArgumentList, name string) string {
	arg := args.ForName(name)
	if arg == nil {
		return ""
	}
	return arg.Value.Raw
}
