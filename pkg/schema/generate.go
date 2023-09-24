package schema

import (
	"github.com/99designs/gqlgen/api"
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/plugin/modelgen"
	"github.com/spf13/afero"
	"github.com/vektah/gqlparser/v2/ast"

	"github.com/roneli/fastgql/pkg/schema/plugin"
	"github.com/roneli/fastgql/pkg/schema/plugin/servergen"
)

func Generate(configPath string, generateServer bool, saveFiles bool, sources ...*ast.Source) error {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return err
	}
	if sources != nil {
		cfg.Sources = append(cfg.Sources, sources...)
	}
	if err := cfg.LoadSchema(); err != nil {
		return err
	}
	// initialize the FastGQL plugin and add it to gqlgen
	fgqlPlugin := NewFastGQLPlugin(cfg.Resolver.Dir())
	srcs, err := fgqlPlugin.CreateAugmented(cfg.Schema)
	if err != nil {
		return err
	}
	// Load config again
	cfg, err = config.LoadConfig(configPath)
	if err != nil {
		return err
	}
	cfg.Sources = srcs
	if saveFiles {
		err = saveGeneratedFiles(srcs)
		if err != nil {
			return err
		}
	}
	// Attaching the mutation function onto modelgen plugin
	modelgenPlugin := modelgen.Plugin{
		MutateHook: plugin.MutateHook,
	}
	// skip validation for now, as after code generation we need to mod tidy again
	cfg.SkipValidation = true
	if generateServer {
		err = api.Generate(cfg, api.NoPlugins(), api.AddPlugin(&modelgenPlugin), api.AddPlugin(New()),
			api.AddPlugin(fgqlPlugin), api.AddPlugin(servergen.New("server.go")))
	} else {
		err = api.Generate(cfg, api.NoPlugins(), api.AddPlugin(&modelgenPlugin),
			api.AddPlugin(New()), api.AddPlugin(fgqlPlugin))
	}
	if err != nil {
		return err
	}
	return nil
}

// saveGeneratedFields saves all the generated files, if the files already exists, it will override them
func saveGeneratedFiles(files []*ast.Source) error {
	fs := afero.NewOsFs()
	for _, file := range files {
		err := afero.WriteFile(fs, file.Name, []byte(file.Input), 0644)
		if err != nil {
			return err
		}
	}
	return nil
}
