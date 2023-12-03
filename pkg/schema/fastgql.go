package schema

import (
	"bytes"
	_ "embed"
	"fmt"
	"github.com/99designs/gqlgen/codegen"
	"github.com/99designs/gqlgen/codegen/templates"
	"go/types"
	"reflect"
	"runtime"
	"strings"
	"text/template"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/vektah/gqlparser/v2/ast"
)

var (
	//go:embed fastgql.tpl
	fastGqlTpl string
	//go:embed fastgql.graphql
	FastGQLSchema     string
	FastGQLDirectives = []string{"table", "generate", "relation", "generateFilterInput", "skipGenerate", "generateMutations", "relation"}
	defaultAugmenters = []Augmenter{
		PaginationAugmenter,
		OrderByAugmenter,
		MutationsAugmenter,
		AggregationAugmenter,
		FilterInputAugmenter,
		FilterArgAugmenter,
	}
)

// FastGqlPlugin augments and extends the original schema
type FastGqlPlugin struct {
	rootDirectory string
}

func NewFastGQLPlugin(rootDir string) FastGqlPlugin {
	return FastGqlPlugin{
		rootDirectory: rootDir,
	}
}

func (f FastGqlPlugin) Name() string {
	return "fastGQLPlugin"
}

func (f FastGqlPlugin) MutateConfig(c *config.Config) error {
	// Skip runtime checks for all FastGQL directives as they only used on the server side schema
	for _, d := range FastGQLDirectives {
		c.Directives[d] = config.DirectiveConfig{SkipRuntime: true}
	}
	return nil
}

func (f FastGqlPlugin) Render(field *codegen.Field) string {
	buf := &bytes.Buffer{}
	if field.TypeReference.Definition.Directives.ForName("generate") != nil {
		return `panic(fmt.Errorf("not implemented"))`
	}
	if field.TypeReference.Definition.IsAbstractType() {
		return `panic(fmt.Errorf("interface not supported"))`
	}
	if field.TypeReference.Definition.IsLeafType() || field.TypeReference.Definition.IsInputType() {
		return `panic(fmt.Errorf("not implemented"))`
	}

	baseFuncs := templates.Funcs()
	baseFuncs["hasSuffix"] = strings.HasSuffix
	baseFuncs["hasPrefix"] = strings.HasPrefix
	baseFuncs["deref"] = deref

	fResolver := fastGQLResolver{field, "postgres"}
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

type fastGQLResolver struct {
	Field   *codegen.Field
	Dialect string
}

// CreateAugmented augments *ast.Schema returning []*ast.Source files that are augmented with filters, mutations etc'
// so gqlgen can generate an augmented fastGQL server
func (f FastGqlPlugin) CreateAugmented(schema *ast.Schema, augmenters ...Augmenter) ([]*ast.Source, error) {
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
