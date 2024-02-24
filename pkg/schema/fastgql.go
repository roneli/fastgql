package schema

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"github.com/spf13/cast"
	"go/types"
	"io/fs"
	"os"
	"reflect"
	"runtime"
	"strings"
	"text/template"

	"github.com/99designs/gqlgen/codegen"
	"github.com/99designs/gqlgen/codegen/templates"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/vektah/gqlparser/v2/ast"
)

var (
	//go:embed fastgql.tpl
	fastGqlTpl string
	//go:embed fastgql.graphql
	FastGQLSchema string
	//go:embed server.gotpl
	fastGqlServerTpl  string
	FastGQLDirectives = []string{"table", "generate", "relation", "generateFilterInput", "isInterfaceFilter", "skipGenerate", "generateMutations", "relation"}
	defaultAugmenters = []Augmenter{
		MutationsAugmenter,
		PaginationAugmenter,
		OrderByAugmenter,
		AggregationAugmenter,
		FilterInputAugmenter,
		FilterArgAugmenter,
	}
)

const defaultResolverTemplate = `
{{ reserveImport "context"  }}
{{ reserveImport "github.com/roneli/fastgql/pkg/execution/builders" }}
{{ reserveImport "github.com/georgysavva/scany/v2/pgxscan" }}
type {{.}} struct {
	Cfg *builders.Config 
	Executor  pgxscan.Querier
}
`

// FastGqlPlugin augments and extends the original schema
type FastGqlPlugin struct {
	rootDirectory  string
	generateServer bool
	serverFilename string
	codgen         *codegen.Data
}

func NewFastGQLPlugin(rootDir, serverFileName string, generateServer bool) *FastGqlPlugin {
	return &FastGqlPlugin{
		rootDirectory:  rootDir,
		generateServer: generateServer,
		serverFilename: serverFileName,
	}
}

func (f *FastGqlPlugin) Name() string {
	return "fastGQLPlugin"
}

func (f *FastGqlPlugin) MutateConfig(c *config.Config) error {
	// Skip runtime checks for all FastGQL directives as they only used on the server side schema
	for _, d := range FastGQLDirectives {
		c.Directives[d] = config.DirectiveConfig{SkipRuntime: true}
	}
	return nil
}

func (f *FastGqlPlugin) GenerateCode(data *codegen.Data) error {
	f.codgen = data
	if _, err := os.Stat(data.Config.Resolver.Filename); errors.Is(err, fs.ErrNotExist) {
		err := templates.Render(templates.Options{
			PackageName: data.Config.Resolver.Package,
			FileNotice: `
				// This file will not be regenerated automatically.
				//
				// It serves as dependency injection for your app, add any dependencies you require here.`,
			Template: defaultResolverTemplate,
			Filename: data.Config.Resolver.Filename,
			Data:     data.Config.Resolver.Type,
			Packages: data.Config.Packages,
		})
		if err != nil {
			return err
		}
	}
	if f.generateServer {
		serverBuild := &struct {
			ExecPackageName     string
			ResolverPackageName string
		}{
			ExecPackageName:     data.Config.Exec.ImportPath(),
			ResolverPackageName: data.Config.Resolver.ImportPath(),
		}

		if _, err := os.Stat(f.serverFilename); os.IsNotExist(err) {
			return templates.Render(templates.Options{
				PackageName: "main",
				Filename:    f.serverFilename,
				Data:        serverBuild,
				Packages:    data.Config.Packages,
				Template:    fastGqlServerTpl,
			})
		}
	}
	return nil
}

func (f *FastGqlPlugin) Implement(field *codegen.Field) string {
	buf := &bytes.Buffer{}
	if field.TypeReference.Definition.Directives.ForName("generate") != nil {
		return `panic(fmt.Errorf("not implemented"))`
	}
	if field.TypeReference.Definition.IsLeafType() || field.TypeReference.Definition.IsInputType() {
		return `panic(fmt.Errorf("not implemented"))`
	}
	baseFuncs := templates.Funcs()
	baseFuncs["hasSuffix"] = strings.HasSuffix
	baseFuncs["hasPrefix"] = strings.HasPrefix
	baseFuncs["deref"] = deref
	var implementors = make(map[string]codegen.InterfaceImplementor)
	var fieldType = field.TypeReference.GO
	var implTypeName = "typename"
	interfaces, ok := f.codgen.Interfaces[field.Type.Name()]
	if ok {
		implTypeName = getTypeName(field.Directives)
		fieldType = interfaces.Type
		for _, implementor := range interfaces.Implementors {
			implementors[implementor.Name] = implementor
		}
	}

	fResolver := fastGQLResolver{field, fieldType, implementors, implTypeName, "postgres"}
	t := template.New("").Funcs(baseFuncs)
	t, err := t.New("fastgql.tpl").Parse(fastGqlTpl)
	if err != nil {
		panic(err)
	}
	if err := t.Execute(buf, fResolver); err != nil {
		panic(err)
	}
	return buf.String()
}

// CreateAugmented augments *ast.Schema returning []*ast.Source files that are augmented with filters, mutations etc'
// so gqlgen can generate an augmented fastGQL server
func (f *FastGqlPlugin) CreateAugmented(schema *ast.Schema, augmenters ...Augmenter) ([]*ast.Source, error) {
	if len(augmenters) == 0 {
		augmenters = defaultAugmenters
	}
	for _, a := range augmenters {
		if err := a(schema); err != nil {
			return nil, fmt.Errorf("augmenter %v failed: %w", GetFunctionName(a), err)
		}
	}
	// Format augmented schema to *.graphql files
	return FormatSchema(f.rootDirectory, schema), nil
}

func GetFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

func ref(p types.Type) string {
	return types.TypeString(p, func(pkg *types.Package) string {
		return pkg.Name()
	})
}

func deref(p types.Type) string {
	return strings.TrimPrefix(ref(p), "*")
}

type fastGQLResolver struct {
	Field                *codegen.Field
	FieldType            types.Type
	Implementors         map[string]codegen.InterfaceImplementor
	ImplementorsTypeName string
	Dialect              string
}

func getTypeName(directives []*codegen.Directive) string {
	for _, d := range directives {
		if d.Name != "typename" {
			continue
		}
		for _, a := range d.Args {
			return cast.ToString(a.Value)
		}
	}
	return "typename"
}
