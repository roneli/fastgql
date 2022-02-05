package resolvergen

import (
	"bytes"
	"go/types"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	"github.com/roneli/fastgql/pkg/codegen/rewrite"

	"github.com/99designs/gqlgen/codegen"
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/99designs/gqlgen/plugin"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
)

func New() plugin.Plugin {
	return &Plugin{}
}

const (
	defaultImpl = `panic(fmt.Errorf("not implemented"))`
	wrapperImpl = `return &{{.Field.TypeReference.GO | deref}}{}, nil`
)

type Plugin struct{}

var _ plugin.CodeGenerator = &Plugin{}

func (m *Plugin) Name() string {
	return "resolvergen"
}

func (m *Plugin) GenerateCode(data *codegen.Data) error {
	if !data.Config.Resolver.IsDefined() {
		return nil
	}

	switch data.Config.Resolver.Layout {
	case config.LayoutSingleFile:
		return m.generateSingleFile(data)
	case config.LayoutFollowSchema:
		return m.generatePerSchema(data)
	}

	return nil
}

func (m *Plugin) generateSingleFile(data *codegen.Data) error {
	file := File{}

	if _, err := os.Stat(data.Config.Resolver.Filename); err == nil {
		// file already exists and we dont support updating codegen with layout = single so just return
		return nil
	}

	for _, o := range data.Objects {
		if o.HasResolvers() {
			file.Objects = append(file.Objects, o)
		}
		for _, f := range o.Fields {
			if !f.IsResolver {
				continue
			}

			resolver := Resolver{o, f, `panic("not implemented")`}
			file.Resolvers = append(file.Resolvers, &resolver)
		}
	}

	resolverBuild := &ResolverBuild{
		File:         &file,
		PackageName:  data.Config.Resolver.Package,
		ResolverType: data.Config.Resolver.Type,
		HasRoot:      true,
	}

	return templates.Render(templates.Options{
		PackageName: data.Config.Resolver.Package,
		FileNotice:  `// THIS CODE IS A STARTING POINT ONLY. IT WILL NOT BE UPDATED WITH SCHEMA CHANGES.`,
		Filename:    data.Config.Resolver.Filename,
		Data:        resolverBuild,
		Packages:    data.Config.Packages,
	})
}

func (m *Plugin) generatePerSchema(data *codegen.Data) error {
	rewriter, err := rewrite.New(data.Config.Resolver.Dir())
	if err != nil {
		return err
	}

	files := map[string]*File{}

	for _, o := range data.Objects {
		if o.HasResolvers() {
			fn := gqlToResolverName(data.Config.Resolver.Dir(), o.Position.Src.Name, data.Config.Resolver.FilenameTemplate)
			if files[fn] == nil {
				files[fn] = &File{}
			}

			rewriter.MarkStructCopied(templates.LcFirst(o.Name) + templates.UcFirst(data.Config.Resolver.Type))
			rewriter.GetMethodBody(data.Config.Resolver.Type, o.Name)
			files[fn].Objects = append(files[fn].Objects, o)
		}
		for _, f := range o.Fields {
			if !f.IsResolver {
				continue
			}

			structName := templates.LcFirst(o.Name) + templates.UcFirst(data.Config.Resolver.Type)
			implementation := strings.TrimSpace(rewriter.GetMethodBody(structName, f.GoFieldName))
			if implementation == "" {
				implementation = `panic(fmt.Errorf("not implemented"))`
			}
			resolver := Resolver{o, f, implementation}
			fn := gqlToResolverName(data.Config.Resolver.Dir(), f.Position.Src.Name, data.Config.Resolver.FilenameTemplate)
			if files[fn] == nil {
				files[fn] = &File{}
			}

			files[fn].Resolvers = append(files[fn].Resolvers, &resolver)
		}
	}

	for filename, file := range files {
		file.imports = rewriter.ExistingImports(filename)
		file.RemainingSource = rewriter.RemainingSource(filename)
	}

	for filename, file := range files {
		resolverBuild := &ResolverBuild{
			File:         file,
			PackageName:  data.Config.Resolver.Package,
			ResolverType: data.Config.Resolver.Type,
		}

		err := templates.Render(templates.Options{
			PackageName: data.Config.Resolver.Package,
			FileNotice: `
				// This file will be automatically regenerated based on the schema, any resolver implementations
				// will be copied through when generating and any unknown code will be moved to the end.`,
			Filename: filename,
			Data:     resolverBuild,
			Packages: data.Config.Packages,
			Funcs: map[string]interface{}{
				"renderResolver": func(resolver interface{}) (*bytes.Buffer, error) {
					r := resolver.(*Resolver)
					return m.renderResolver(r)
				},
			},
		})
		if err != nil {
			return err
		}
	}

	if _, err := os.Stat(data.Config.Resolver.Filename); os.IsNotExist(errors.Cause(err)) {
		err := templates.Render(templates.Options{
			PackageName: data.Config.Resolver.Package,
			FileNotice: `
				// This file will not be regenerated automatically.
				//
				// It serves as dependency injection for your app, add any dependencies you require here.`,
			Template: `{{ reserveImport "context"  }}
{{ reserveImport "github.com/roneli/fastgql/builders" }}
{{ reserveImport "github.com/roneli/fastgql/execution" }}
type {{.}} struct {
	Cfg *builders.Config 
	Executor execution.Querier
}`,
			Filename: data.Config.Resolver.Filename,
			Data:     data.Config.Resolver.Type,
			Packages: data.Config.Packages,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Plugin) renderResolver(resolver *Resolver) (*bytes.Buffer, error) {
	buf := &bytes.Buffer{}
	if resolver.Implementation != "" && defaultImpl != resolver.Implementation {
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

	if d := resolver.Field.FieldDefinition.Directives.ForName("generate"); d != nil {
		if v := d.Arguments.ForName("wrapper"); v != nil && cast.ToBool(v.Value.Raw) {
			t, err := template.New("").Funcs(baseFuncs).Parse(wrapperImpl)
			if err != nil {
				return buf, err
			}

			return buf, t.Execute(buf, resolver)
		}
	}

	if d := resolver.Field.FieldDefinition.Directives.ForName("skipGenerate"); d != nil {
		if v := d.Definition.Arguments.ForName("resolver"); v != nil && cast.ToBool(v.DefaultValue.Raw) {
			buf.WriteString(`panic(fmt.Errorf("not implemented"))`)
			return buf, nil
		}
	}

	t := template.New("").Funcs(baseFuncs)
	fileName := resolveName("sql.tpl", 0)

	b, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	t, err = t.New(filepath.Base(fileName)).Parse(string(b))
	if err != nil {
		panic(err)
	}

	return buf, t.Execute(buf, resolver)
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

type ResolverBuild struct {
	*File
	HasRoot      bool
	PackageName  string
	ResolverType string
}

type File struct {
	// These are separated because the type definition of the resolver object may live in a different file from the
	// resolver method implementations, for example when extending a type in a different graphql schema file
	Objects         []*codegen.Object
	Resolvers       []*Resolver
	imports         []rewrite.Import
	RemainingSource string
}

func (f *File) Imports() string {
	for _, imp := range f.imports {
		if imp.Alias == "" {
			_, _ = templates.CurrentImports.Reserve(imp.ImportPath)
		} else {
			_, _ = templates.CurrentImports.Reserve(imp.ImportPath, imp.Alias)
		}
	}
	return ""
}

type Resolver struct {
	Object         *codegen.Object
	Field          *codegen.Field
	Implementation string
}

func gqlToResolverName(base string, gqlname, filenameTmpl string) string {
	gqlname = filepath.Base(gqlname)
	ext := filepath.Ext(gqlname)
	if filenameTmpl == "" {
		filenameTmpl = "{name}.fastgql.go"
	}
	filename := strings.ReplaceAll(filenameTmpl, "{name}", strings.TrimSuffix(gqlname, ext))
	return filepath.Join(base, filename)
}
