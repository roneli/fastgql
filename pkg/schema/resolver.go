package schema

import (
	"bytes"
	_ "embed"
	"fmt"
	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/roneli/fastgql/pkg/schema/plugin/resolvergen"
	"github.com/spf13/cast"
	"go/types"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
)

var (
	//go:embed resolver.gotpl
	resolverTemplate string
	//go:embed fastgql.tpl
	fastGqlTpl string
)

const (
	wrapperImpl = `return &{{.Field.TypeReference.GO | deref}}{}, nil`
)

type ResolverPlugin struct {
	resolvergen.Plugin
}

func New() *ResolverPlugin {
	return &ResolverPlugin{
		Plugin: resolvergen.Plugin{
			ExtraFuncs: template.FuncMap{
				"renderResolver": renderResolver,
			},
			ResolverTemplate: resolverTemplate,
		},
	}
}

type fastGQLResolver struct {
	*resolvergen.Resolver
	Dialect string
}

func renderResolver(resolver *resolvergen.Resolver) (*bytes.Buffer, error) {
	buf := &bytes.Buffer{}
	defaultImpl := fmt.Sprintf("panic(fmt.Errorf(\"not implemented: %v - %v\"))", resolver.Field.GoFieldName, resolver.Field.Name)
	if resolver.Implementation != "" && resolver.Implementation != defaultImpl {
		buf.WriteString(resolver.Implementation)
		return buf, nil
	}

	if resolver.Field.TypeReference.Definition.IsAbstractType() {
		buf.WriteString(`panic(fmt.Errorf("interface support not implemented"))`)
		return buf, nil
	}

	if resolver.Field.TypeReference.Definition.IsLeafType() || resolver.Field.TypeReference.Definition.IsInputType() {
		buf.WriteString(`panic(fmt.Errorf("not implemented"))`)
		return buf, nil
	}
	baseFuncs := templates.Funcs()
	baseFuncs["hasSuffix"] = strings.HasSuffix
	baseFuncs["hasPrefix"] = strings.HasPrefix
	baseFuncs["deref"] = deref
	fResolver := fastGQLResolver{resolver, "postgres"}
	if d := resolver.Field.TypeReference.Definition.Directives.ForName("generate"); d != nil {
		if v := d.Arguments.ForName("wrapper"); v != nil && cast.ToBool(v.Value.Raw) {
			t, err := template.New("").Funcs(baseFuncs).Parse(wrapperImpl)
			if err != nil {
				return buf, err
			}
			return buf, t.Execute(buf, fResolver)
		}
	}

	if d := resolver.Field.FieldDefinition.Directives.ForName("skipGenerate"); d != nil {
		if v := d.Definition.Arguments.ForName("resolver"); v != nil && cast.ToBool(v.DefaultValue.Raw) {
			buf.WriteString(`panic(fmt.Errorf("not implemented"))`)
			return buf, nil
		}
	}

	if d := resolver.Field.TypeReference.Definition.Directives.ForName("dialect"); d != nil {
		if v := d.Arguments.ForName("type"); v != nil && v.Value.String() != "" {
			fResolver.Dialect = v.Value.Raw
		}
	}
	t := template.New("").Funcs(baseFuncs)
	t, err := t.New("fastgql.tpl").Parse(fastGqlTpl)
	if err != nil {
		panic(err)
	}
	return buf, t.Execute(buf, fResolver)
}

func deref(p types.Type) string {
	return strings.TrimPrefix(templates.CurrentImports.LookupType(p), "*")
}

func resolveName(name string, skip int) string {
	if name[0] == '.' {
		// load path relative to calling source file
		_, callerFile, _, _ := runtime.Caller(skip + 1)
		return filepath.Join(filepath.Dir(callerFile), name[1:])
	}

	// load path relative to this directory
	_, callerFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(callerFile), name)
}
