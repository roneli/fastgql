package plugin

import "github.com/99designs/gqlgen/plugin/modelgen"

// Defining mutation function
func MutateHook(b *modelgen.ModelBuild) *modelgen.ModelBuild {
	for _, model := range b.Models {
		for _, field := range model.Fields {
			field.Tag += ` db:"` + field.Name + `"`
		}
	}

	return b
}
