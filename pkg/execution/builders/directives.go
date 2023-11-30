package builders

import (
	"fmt"
	"github.com/roneli/fastgql/pkg/schema"

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
		Fields:               cast.ToStringSlice(schema.GetDirectiveValue(d, "fields")),
		References:           cast.ToStringSlice(schema.GetDirectiveValue(d, "references")),
		BaseTable:            cast.ToString(schema.GetDirectiveValue(d, "baseTable")),
		ReferenceTable:       cast.ToString(schema.GetDirectiveValue(d, "refTable")),
		ManyToManyTable:      cast.ToString(schema.GetDirectiveValue(d, "manyToManyTable")),
		ManyToManyFields:     cast.ToStringSlice(schema.GetDirectiveValue(d, "manyToManyFields")),
		ManyToManyReferences: cast.ToStringSlice(schema.GetDirectiveValue(d, "manyToManyReferences")),
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
