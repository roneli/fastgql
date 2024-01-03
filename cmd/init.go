package cmd

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/roneli/fastgql/pkg/schema"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/spf13/cobra"
)

var configTemplate = template.Must(template.New("name").Parse(
	`# Where are all the schema files located? globs are supported eg  src/**/*.graphqls
schema:
  - graph/*.graphql
# Where should the generated servergen code go?
exec:
  filename: graph/generated/generated.go
  package: generated
# Uncomment to enable federation
# federation:
#   filename: graph/generated/federation.go
#   package: generated
# Where should any generated models go?
model:
  filename: graph/model/models_gen.go
  package: model
# Where should the resolver implementations go?
resolver:
  layout: follow-schema
  dir: graph
  package: graph
# Optional: turn on use ` + "`" + `gqlgen:"fieldName"` + "`" + ` tags in your models
# struct_tag: json
# Optional: turn on to use []Thing instead of []*Thing
# omit_slice_element_pointers: false
# Optional: set to speed up generation time by not performing a final validation pass.
# skip_validation: true
# gqlgen will search for any type names in the schema in these go packages
# if they match it will use them, otherwise it will generate them.
autobind:
  - "{{.}}/graph/model"
# This section declares type mapping between the GraphQL and go type systems
#
# The first line in each type will be used as defaults for resolver arguments and
# modelgen, the others will be allowed when binding to fields. Configure them to
# your liking
models:
  Id:
    model:
      - github.com/99designs/gqlgen/graphql.Id
      - github.com/99designs/gqlgen/graphql.Int
      - github.com/99designs/gqlgen/graphql.Int64
      - github.com/99designs/gqlgen/graphql.Int32
  Int:
    model:
      - github.com/99designs/gqlgen/graphql.Int
      - github.com/99designs/gqlgen/graphql.Int64
      - github.com/99designs/gqlgen/graphql.Int32
`))

const schemaDefault = `type User @table(name: "user"){
  id: Int!
  name: String!
  posts: [Post] @relation(type: ONE_TO_MANY, fields: ["id"], references: ["user_id"])
}

type Post @generateFilterInput {
  id: Int!
  name: String
  categories: [Category] @relation(type: MANY_TO_MANY, fields: ["id"], references: ["id"], 
	manyToManyTable: "posts_to_categories", manyToManyFields: ["post_id"], manyToManyReferences: ["category_id"])
  user_id: Int
  user: User @relation(type: ONE_TO_ONE, fields: ["user_id"], references: ["id"])
}


type Category @generateFilterInput{
  id: Int!
  name: String
}

type Query {
  posts: [Post] @generate
  users: [User] @generate 
  categories: [Category] @generate
}
`

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "create a new fastgql project in current directory",
	Long:  `Generates a start fastgql project with servergen, resolvers and schema ready`,
	RunE: func(cmd *cobra.Command, args []string) error {
		pkgName := schema.ImportPathForDir(".")
		if pkgName == "" {
			return errors.New("unable to determine import path for current directory, you probably need to run go mod init first")
		}

		if err := initSchema(schemaFilename); err != nil {
			return err
		}
		if err := initFastgqlSchema(); err != nil {
			return err
		}
		if err := initModelFile(); err != nil {
			return err
		}
		if !configExists(configFilename) {
			if err := initConfig(configFilename, pkgName); err != nil {
				return err
			}
		}
		if err := schema.Generate(configFilename, true, false); err != nil {
			fmt.Fprintln(os.Stderr, "failed to load config", err.Error())
			os.Exit(2)
		}
		return nil
	},
}

func configExists(configFilename string) bool {
	var cfg *config.Config
	if configFilename != "" {
		cfg, _ = config.LoadConfig(configFilename)
	} else {
		cfg, _ = config.LoadConfigFromDefaultLocations()
	}
	return cfg != nil
}

func initConfig(configFilename string, pkgName string) error {
	if configFilename == "" {
		configFilename = "gqlgen.yml"
	}

	if err := os.MkdirAll(filepath.Dir(configFilename), 0755); err != nil {
		return fmt.Errorf("unable to create config dir: " + err.Error())
	}

	var buf bytes.Buffer
	if err := configTemplate.Execute(&buf, pkgName); err != nil {
		panic(err)
	}

	if err := os.WriteFile(configFilename, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("unable to write cfg file: " + err.Error())
	}

	return nil
}

func initModelFile() error {
	if err := os.MkdirAll(filepath.Dir("graph/model/models_fastgql.go"), 0755); err != nil {
		return fmt.Errorf("unable to create model dir: " + err.Error())
	}
	if err := os.WriteFile("graph/model/models_fastgql.go", []byte("package model"), 0644); err != nil {
		return fmt.Errorf("unable to write model file: " + err.Error())
	}
	return nil
}

func initFastgqlSchema() error {
	if err := os.MkdirAll(filepath.Dir("graph/fastgql.graphql"), 0755); err != nil {
		return fmt.Errorf("unable to create schema dir: " + err.Error())
	}

	if err := os.WriteFile("graph/fastgql.graphql", []byte(strings.TrimSpace(schema.FastGQLSchema)), 0644); err != nil {
		return fmt.Errorf("unable to write schema file: " + err.Error())
	}
	return nil
}

func initSchema(schemaFilename string) error {
	schemaFullPath := filepath.Join("graph", schemaFilename)
	_, err := os.Stat(schemaFullPath)
	if !os.IsNotExist(err) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(schemaFullPath), 0755); err != nil {
		return fmt.Errorf("unable to create schema dir: " + err.Error())
	}

	if err = os.WriteFile(schemaFullPath, []byte(strings.TrimSpace(schemaDefault)), 0644); err != nil {
		return fmt.Errorf("unable to write schema file: " + err.Error())
	}
	return nil
}
