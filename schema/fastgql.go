package schema

import (
	"bytes"
	"fmt"
	"github.com/99designs/gqlgen/api"
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/plugin/modelgen"
	"github.com/roneli/fastgql/plugin/resolvergen"
	"github.com/roneli/fastgql/plugin/servergen"
	"github.com/roneli/fastgql/schema/augmenters"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/formatter"
)



type FastGqlPlugin struct {}

func (f FastGqlPlugin) Name() string {
	return "fastGQLPlugin"
}

func (f FastGqlPlugin) MutateConfig(cfg *config.Config) error {
	cfg.Directives["generateArguments"] = config.DirectiveConfig{SkipRuntime: true}
	cfg.Directives["generateFilterInput"] = config.DirectiveConfig{SkipRuntime: true}
	cfg.Directives["sqlRelation"] = config.DirectiveConfig{SkipRuntime: true}
	cfg.Directives["tableName"] = config.DirectiveConfig{SkipRuntime: true}
	return nil
}

// TODO: make this configurable
func (f FastGqlPlugin) CreateAugmented(schema *ast.Schema) *ast.Source {
	_ = augmenters.Pagination{}.Augment(schema)
	_ = augmenters.Ordering{}.Augment(schema)
	_ = augmenters.FilterInput{}.Augment(schema)
	_ = augmenters.FilterArguments{}.Augment(schema)

	var buf bytes.Buffer
	formatter.NewFormatter(&buf).FormatSchema(schema)
	fmt.Print(buf.String())
	return &ast.Source{
		Name:    "schema.graphql",
		Input:   buf.String(),
		BuiltIn: false,
	}
}


func Generate(configPath string, generateServer bool) error {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return err
	}

	err = cfg.LoadSchema()
	if err != nil {
		return err
	}
	plugin := FastGqlPlugin{}
	src := plugin.CreateAugmented(cfg.Schema)

	// Load config again
	cfg, err = config.LoadConfig(configPath)
	if err != nil {
		return err
	}
	cfg.Sources = []*ast.Source{src}

	if generateServer {
		err = api.Generate(cfg,  api.NoPlugins(), api.AddPlugin(modelgen.New()), api.AddPlugin(resolvergen.New()),
			api.AddPlugin(plugin), api.AddPlugin(servergen.New("server.go")))
	} else {
		err = api.Generate(cfg,  api.NoPlugins(), api.AddPlugin(modelgen.New()), api.AddPlugin(resolvergen.New()), api.AddPlugin(plugin))
	}

	if err != nil {
		return err
	}
	return nil
}