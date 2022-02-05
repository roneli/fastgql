package schema

import (
	"bytes"
	_ "embed"

	augmenters2 "github.com/roneli/fastgql/pkg/schema/augmenters"
	"github.com/roneli/fastgql/pkg/schema/plugin"
	"github.com/roneli/fastgql/pkg/schema/plugin/resolvergen"
	"github.com/roneli/fastgql/pkg/schema/plugin/servergen"

	"github.com/99designs/gqlgen/api"
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/plugin/modelgen"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/formatter"
)

//go:embed fastgql.graphql
var FastGQLSchema string

type FastGqlPlugin struct{}

func (f FastGqlPlugin) Name() string {
	return "fastGQLPlugin"
}

func (f FastGqlPlugin) MutateConfig(c *config.Config) error {
	c.Directives["generateArguments"] = config.DirectiveConfig{SkipRuntime: true}
	c.Directives["generateFilterInput"] = config.DirectiveConfig{SkipRuntime: true}
	c.Directives["sqlRelation"] = config.DirectiveConfig{SkipRuntime: true}
	c.Directives["tableName"] = config.DirectiveConfig{SkipRuntime: true}
	c.Directives["generateMutations"] = config.DirectiveConfig{SkipRuntime: true}
	c.Directives["generate"] = config.DirectiveConfig{SkipRuntime: true}

	for _, schemaType := range c.Schema.Types {
		if schemaType == c.Schema.Query || schemaType == c.Schema.Mutation || schemaType == c.Schema.Subscription {
			continue
		}
		if rg := schemaType.Directives.ForName("generate"); rg != nil {
			for _, f := range schemaType.Fields {
				if c.Models[schemaType.Name].Fields == nil {
					c.Models[schemaType.Name] = config.TypeMapEntry{
						Model:  c.Models[schemaType.Name].Model,
						Fields: map[string]config.TypeMapField{},
					}
				}

				c.Models[schemaType.Name].Fields[f.Name] = config.TypeMapField{
					FieldName: f.Name,
					Resolver:  true,
				}
			}
		}

	}

	return nil
}

// TODO: make this configurable
func (f FastGqlPlugin) CreateAugmented(schema *ast.Schema) *ast.Source {
	_ = augmenters2.Pagination{}.Augment(schema)
	_ = augmenters2.Ordering{}.Augment(schema)
	_ = augmenters2.Aggregation{}.Augment(schema)
	_ = augmenters2.FilterInput{}.Augment(schema)
	_ = augmenters2.FilterArguments{}.Augment(schema)
	_ = augmenters2.Mutations{}.Augment(schema)

	var buf bytes.Buffer
	formatter.NewFormatter(&buf).FormatSchema(schema)

	return &ast.Source{
		Name:    "schema.graphql",
		Input:   buf.String(),
		BuiltIn: false,
	}
}

func Generate(configPath string, generateServer bool, sources ...*ast.Source) error {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return err
	}
	if sources != nil {
		cfg.Sources = append(cfg.Sources, sources...)
	}
	err = cfg.LoadSchema()
	if err != nil {
		return err
	}
	fgqlPlugin := FastGqlPlugin{}
	src := fgqlPlugin.CreateAugmented(cfg.Schema)

	// Load config again
	cfg, err = config.LoadConfig(configPath)
	if err != nil {
		return err
	}
	cfg.Sources = []*ast.Source{src}

	// Attaching the mutation function onto modelgen plugin
	modelgenPlugin := modelgen.Plugin{
		MutateHook: plugin.MutateHook,
	}

	if generateServer {
		err = api.Generate(cfg, api.NoPlugins(), api.AddPlugin(&modelgenPlugin), api.AddPlugin(resolvergen.New()),
			api.AddPlugin(fgqlPlugin), api.AddPlugin(servergen.New("server.go")))
	} else {
		err = api.Generate(cfg, api.NoPlugins(), api.AddPlugin(&modelgenPlugin), api.AddPlugin(resolvergen.New()), api.AddPlugin(fgqlPlugin))
	}

	if err != nil {
		return err
	}
	return nil
}
