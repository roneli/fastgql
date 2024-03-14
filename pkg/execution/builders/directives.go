package builders

import (
	"fmt"

	"github.com/vektah/gqlparser/v2/ast"
)

type DialectDirective struct {
	Dialect       string
	ParentDialect string
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
