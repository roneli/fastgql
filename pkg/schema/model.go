package schema

import (
	"github.com/99designs/gqlgen/plugin/modelgen"
	"github.com/iancoleman/strcase"
)

// Defining mutation function
func mutateHook(b *modelgen.ModelBuild) *modelgen.ModelBuild {
	for _, model := range b.Models {
		for _, field := range model.Fields {
			field.Tag += ` db:"` + strcase.ToSnake(field.Name) + `"`
		}
	}
	return b
}
