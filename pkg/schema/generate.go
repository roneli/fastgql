package schema

import (
	"fmt"
	"log"

	"github.com/99designs/gqlgen/plugin/modelgen"

	"github.com/99designs/gqlgen/api"
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/spf13/afero"
	"github.com/vektah/gqlparser/v2/ast"
)

// Generate generates the schema and the resolver files, if generateServer is true, it will also generate the server file.
// if saveFiles is true, it will save the generated augmented graphql files to the disk, otherwise it the only be saved in generated code.
func Generate(configPath string, generateServer, saveFiles bool, serverFilename string, sources ...*ast.Source) error {
	log.Printf("loading config from %s", configPath)
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config in %s: %w", configPath, err)
	}
	if sources != nil {
		cfg.Sources = append(cfg.Sources, sources...)
	}
	if err := cfg.LoadSchema(); err != nil {
		return err
	}
	// initialize the FastGQL plugin and add it to gqlgen
	if serverFilename == "" {
		serverFilename = "server.go"
	}
	fgqlPlugin := NewFastGQLPlugin(cfg.Resolver.Package, serverFilename, generateServer)
	srcs, err := fgqlPlugin.CreateAugmented(cfg.Schema)
	if err != nil {
		return err
	}
	log.Print("augmented schema generated successfully")
	// Load config again
	log.Printf("loading config again from %s", configPath)
	cfg, err = config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config in %s: %w", configPath, err)
	}
	cfg.Sources = srcs
	if saveFiles {
		if err := saveGeneratedFiles(srcs); err != nil {
			return err
		}
	}
	// Attaching the mutation function onto modelgen plugin
	p := modelgen.Plugin{
		MutateHook: mutateHook,
	}
	// skip validation for now, as after code generation we need to mod tidy again
	cfg.SkipValidation = true
	if err = api.Generate(cfg, api.PrependPlugin(fgqlPlugin), api.ReplacePlugin(&p)); err != nil {
		return err
	}
	log.Print("fastgql generated successfully")
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
