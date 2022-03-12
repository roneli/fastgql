package builders

import "github.com/99designs/gqlgen/codegen/config"

func FilterHook(cfg config.Config, field Field) error {

	filter := field.Arguments["filter"]

}
