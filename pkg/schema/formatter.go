package schema

import (
	"bytes"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"
)

// Save all fastgql augmented schema into fastgql_schema.graphql file.
const defaultFastGqlSchema = "fastgql_schema.graphql"

// Format into multiple source, the original format schema from gqlparser lib saves all in one file,
// in this case after augmentation we want to keep all original files and structure and all added definitions to put in
// fastgql_schema.graphql file.
func Format(resolverPackageDir string, schema *ast.Schema) []*ast.Source {
	if schema == nil {
		return nil
	}
	defaultFormatter := newFormatter(&bytes.Buffer{})
	defaultSource := path.Join(resolverPackageDir, defaultFastGqlSchema)
	formatters := map[string]*formatter{
		defaultSource: defaultFormatter,
	}
	var inSchema bool
	startSchema := func() {
		if !inSchema {
			inSchema = true

			defaultFormatter.WriteWord("schema").WriteString("{").WriteNewline()
			defaultFormatter.IncrementIndent()
		}
	}
	if schema.Query != nil && schema.Query.Name != "Query" {
		startSchema()
		defaultFormatter.WriteWord("query").NoPadding().WriteString(":").NeedPadding()
		defaultFormatter.WriteWord(schema.Query.Name).WriteNewline()
	}
	if schema.Mutation != nil && schema.Mutation.Name != "Mutation" {
		startSchema()
		defaultFormatter.WriteWord("mutation").NoPadding().WriteString(":").NeedPadding()
		defaultFormatter.WriteWord(schema.Mutation.Name).WriteNewline()
	}
	if schema.Subscription != nil && schema.Subscription.Name != "Subscription" {
		startSchema()
		defaultFormatter.WriteWord("subscription").NoPadding().WriteString(":").NeedPadding()
		defaultFormatter.WriteWord(schema.Subscription.Name).WriteNewline()
	}
	if inSchema {
		defaultFormatter.DecrementIndent()
		defaultFormatter.WriteString("}").WriteNewline()
	}

	directiveNames := make([]string, 0, len(schema.Directives))
	for name := range schema.Directives {
		directiveNames = append(directiveNames, name)
	}
	sort.Strings(directiveNames)
	for _, name := range directiveNames {
		d := schema.Directives[name]
		f := getOrCreateFormatter(getSourceName(d.Position, defaultSource), formatters)
		f.FormatDirectiveDefinition(d)
	}

	typeNames := make([]string, 0, len(schema.Types))
	for name := range schema.Types {
		typeNames = append(typeNames, name)
	}
	sort.Strings(typeNames)

	// Collect extension fields grouped by source file
	// Map: source file -> type name -> fields
	extensionFields := make(map[string]map[string]*extensionInfo)

	for _, name := range typeNames {
		t := schema.Types[name]
		sourceName := getSourceName(t.Position, defaultSource)
		f := getOrCreateFormatter(sourceName, formatters)
		// Set the expected source so extended fields are filtered out
		f.expectedSource = t.Position
		f.FormatDefinition(t, false)
		f.expectedSource = nil

		// Collect extension fields for this type
		collectExtensionFields(t, sourceName, extensionFields)
	}

	// Write extension statements to their source files
	for extSourceName, typeExtensions := range extensionFields {
		f := getOrCreateFormatter(extSourceName, formatters)
		for _, extInfo := range typeExtensions {
			f.FormatExtendDefinition(extInfo)
		}
	}

	sources := make([]*ast.Source, 0, len(formatters))
	for name, f := range formatters {
		// skip empty
		if f.writer.Len() == 0 {
			continue
		}
		sources = append(sources, &ast.Source{Name: name, Input: f.writer.String()})
	}
	return sources
}

// extensionInfo holds information about fields extending a type from a different source
type extensionInfo struct {
	TypeName string
	Kind     ast.DefinitionKind
	Fields   ast.FieldList
}

// collectExtensionFields finds fields that came from a different source than the type definition
func collectExtensionFields(t *ast.Definition, typeSourceName string, extensions map[string]map[string]*extensionInfo) {
	if t.Position == nil || t.Position.Src == nil {
		return
	}

	for _, field := range t.Fields {
		if field.Position == nil || field.Position.Src == nil {
			continue
		}
		// If field's source differs from type's source, it's from an extension
		fieldSourceName := field.Position.Src.Name
		if fieldSourceName != t.Position.Src.Name {
			if extensions[fieldSourceName] == nil {
				extensions[fieldSourceName] = make(map[string]*extensionInfo)
			}
			if extensions[fieldSourceName][t.Name] == nil {
				extensions[fieldSourceName][t.Name] = &extensionInfo{
					TypeName: t.Name,
					Kind:     t.Kind,
					Fields:   ast.FieldList{},
				}
			}
			extensions[fieldSourceName][t.Name].Fields = append(extensions[fieldSourceName][t.Name].Fields, field)
		}
	}
}

func newFormatter(w *bytes.Buffer) *formatter {
	return &formatter{writer: w}
}

// formatter is copied from https://github.com/vektah/gqlparser/blob/master/formatter/formatter.go
type formatter struct {
	writer *bytes.Buffer

	indent      int
	emitBuiltin bool

	padNext  bool
	lineHead bool

	// expectedSource is used to filter out fields that came from a different source
	// when formatting a definition. This is set to the position of the type definition
	// and reset to nil after formatting.
	expectedSource *ast.Position
}

func (f *formatter) writeString(s string) {
	_, _ = f.writer.Write([]byte(s))
}

func (f *formatter) writeIndent() *formatter {
	if f.lineHead {
		f.writeString(strings.Repeat("\t", f.indent))
	}
	f.lineHead = false
	f.padNext = false

	return f
}

func (f *formatter) WriteNewline() *formatter {
	f.writeString("\n")
	f.lineHead = true
	f.padNext = false

	return f
}

func (f *formatter) WriteWord(word string) *formatter {
	if f.lineHead {
		f.writeIndent()
	}
	if f.padNext {
		f.writeString(" ")
	}
	f.writeString(strings.TrimSpace(word))
	f.padNext = true

	return f
}

func (f *formatter) WriteString(s string) *formatter {
	if f.lineHead {
		f.writeIndent()
	}
	if f.padNext {
		f.writeString(" ")
	}
	f.writeString(s)
	f.padNext = false

	return f
}

func (f *formatter) WriteDescription(s string) *formatter {
	if s == "" {
		return f
	}

	f.WriteString(`"""`).WriteNewline()

	ss := strings.Split(s, "\n")
	for _, s := range ss {
		f.WriteString(s).WriteNewline()
	}

	f.WriteString(`"""`).WriteNewline()

	return f
}

func (f *formatter) IncrementIndent() {
	f.indent++
}

func (f *formatter) DecrementIndent() {
	f.indent--
}

func (f *formatter) NoPadding() *formatter {
	f.padNext = false

	return f
}

func (f *formatter) NeedPadding() *formatter {
	f.padNext = true

	return f
}

func (f *formatter) FormatOperationTypeDefinition(def *ast.OperationTypeDefinition) {
	f.WriteWord(string(def.Operation)).NoPadding().WriteString(":").NeedPadding()
	f.WriteWord(def.Type)
	f.WriteNewline()
}

func (f *formatter) FormatFieldList(fieldList ast.FieldList) {
	if len(fieldList) == 0 {
		return
	}

	// Filter fields based on expected source (to exclude extension fields)
	var filteredFields ast.FieldList
	for _, field := range fieldList {
		if f.shouldIncludeField(field) {
			filteredFields = append(filteredFields, field)
		}
	}

	if len(filteredFields) == 0 {
		return
	}

	f.WriteString("{").WriteNewline()
	f.IncrementIndent()

	for _, field := range filteredFields {
		f.FormatFieldDefinition(field)
	}

	f.DecrementIndent()
	f.WriteString("}")
}

// shouldIncludeField returns true if the field should be included in the current output.
// Fields from extensions (different source file) are excluded when expectedSource is set.
func (f *formatter) shouldIncludeField(field *ast.FieldDefinition) bool {
	// If no expected source is set, include all fields
	if f.expectedSource == nil || f.expectedSource.Src == nil {
		return true
	}
	// If field has no position info, include it (augmented fields)
	if field.Position == nil || field.Position.Src == nil {
		return true
	}
	// Include field only if it's from the same source as the type definition
	return field.Position.Src.Name == f.expectedSource.Src.Name
}

func (f *formatter) FormatFieldDefinition(field *ast.FieldDefinition) {
	if !f.emitBuiltin && strings.HasPrefix(field.Name, "__") {
		return
	}

	f.WriteDescription(field.Description)

	f.WriteWord(field.Name).NoPadding()
	f.FormatArgumentDefinitionList(field.Arguments)
	f.NoPadding().WriteString(":").NeedPadding()
	f.FormatType(field.Type)

	if field.DefaultValue != nil {
		f.WriteWord("=")
		f.FormatValue(field.DefaultValue)
	}

	f.FormatDirectiveList(field.Directives)

	f.WriteNewline()
}

func (f *formatter) FormatArgumentDefinitionList(lists ast.ArgumentDefinitionList) {
	if len(lists) == 0 {
		return
	}

	f.WriteString("(")
	for idx, arg := range lists {
		f.FormatArgumentDefinition(arg)
		if idx != len(lists)-1 {
			f.NoPadding().WriteWord(",")
		}
	}
	f.NoPadding().WriteString(")").NeedPadding()
}

func (f *formatter) FormatArgumentDefinition(def *ast.ArgumentDefinition) {
	if def.Description != "" {
		f.WriteNewline().IncrementIndent()
		f.WriteDescription(def.Description)
	}

	f.WriteWord(def.Name).NoPadding().WriteString(":").NeedPadding()
	f.FormatType(def.Type)

	if def.DefaultValue != nil {
		f.WriteWord("=")
		f.FormatValue(def.DefaultValue)
	}

	if def.Description != "" {
		f.DecrementIndent()
	}
}

func (f *formatter) FormatDirectiveLocation(location ast.DirectiveLocation) {
	f.WriteWord(string(location))
}

func (f *formatter) FormatDirectiveDefinitionList(lists ast.DirectiveDefinitionList) {
	if len(lists) == 0 {
		return
	}

	for _, dec := range lists {
		f.FormatDirectiveDefinition(dec)
	}
}

func (f *formatter) FormatDirectiveDefinition(def *ast.DirectiveDefinition) {
	if !f.emitBuiltin {
		if def.Position.Src.BuiltIn {
			return
		}
	}

	f.WriteDescription(def.Description)
	f.WriteWord("directive").WriteString("@").WriteWord(def.Name)

	if len(def.Arguments) != 0 {
		f.NoPadding()
		f.FormatArgumentDefinitionList(def.Arguments)
	}

	if len(def.Locations) != 0 {
		f.WriteWord("on")

		for idx, dirLoc := range def.Locations {
			f.FormatDirectiveLocation(dirLoc)

			if idx != len(def.Locations)-1 {
				f.WriteWord("|")
			}
		}
	}

	f.WriteNewline()
}

func (f *formatter) FormatDefinitionList(lists ast.DefinitionList, extend bool) {
	if len(lists) == 0 {
		return
	}

	for _, dec := range lists {
		f.FormatDefinition(dec, extend)
	}
}

func (f *formatter) FormatDefinition(def *ast.Definition, extend bool) {
	if !f.emitBuiltin && def.BuiltIn {
		return
	}

	f.WriteDescription(def.Description)

	if extend {
		f.WriteWord("extend")
	}

	switch def.Kind {
	case ast.Scalar:
		f.WriteWord("scalar").WriteWord(def.Name)

	case ast.Object:
		f.WriteWord("type").WriteWord(def.Name)

	case ast.Interface:
		f.WriteWord("interface").WriteWord(def.Name)

	case ast.Union:
		f.WriteWord("union").WriteWord(def.Name)

	case ast.Enum:
		f.WriteWord("enum").WriteWord(def.Name)

	case ast.InputObject:
		f.WriteWord("input").WriteWord(def.Name)
	}

	if len(def.Interfaces) != 0 {
		f.WriteWord("implements").WriteWord(strings.Join(def.Interfaces, " & "))
	}

	f.FormatDirectiveList(def.Directives)

	if len(def.Types) != 0 {
		f.WriteWord("=").WriteWord(strings.Join(def.Types, " | "))
	}

	f.FormatFieldList(def.Fields)

	f.FormatEnumValueList(def.EnumValues)

	f.WriteNewline()
}

func (f *formatter) FormatEnumValueList(lists ast.EnumValueList) {
	if len(lists) == 0 {
		return
	}

	f.WriteString("{").WriteNewline()
	f.IncrementIndent()

	for _, v := range lists {
		f.FormatEnumValueDefinition(v)
	}

	f.DecrementIndent()
	f.WriteString("}")
}

func (f *formatter) FormatEnumValueDefinition(def *ast.EnumValueDefinition) {
	f.WriteDescription(def.Description)

	f.WriteWord(def.Name)
	f.FormatDirectiveList(def.Directives)

	f.WriteNewline()
}

func (f *formatter) FormatOperationList(lists ast.OperationList) {
	for _, def := range lists {
		f.FormatOperationDefinition(def)
	}
}

func (f *formatter) FormatOperationDefinition(def *ast.OperationDefinition) {
	f.WriteWord(string(def.Operation))
	if def.Name != "" {
		f.WriteWord(def.Name)
	}
	f.FormatVariableDefinitionList(def.VariableDefinitions)
	f.FormatDirectiveList(def.Directives)

	if len(def.SelectionSet) != 0 {
		f.FormatSelectionSet(def.SelectionSet)
		f.WriteNewline()
	}
}

func (f *formatter) FormatDirectiveList(lists ast.DirectiveList) {
	if len(lists) == 0 {
		return
	}

	for _, dir := range lists {
		f.FormatDirective(dir)
	}
}

func (f *formatter) FormatDirective(dir *ast.Directive) {
	f.WriteString("@").WriteWord(dir.Name)
	f.FormatArgumentList(dir.Arguments)
}

func (f *formatter) FormatArgumentList(lists ast.ArgumentList) {
	if len(lists) == 0 {
		return
	}
	f.NoPadding().WriteString("(")
	for idx, arg := range lists {
		f.FormatArgument(arg)

		if idx != len(lists)-1 {
			f.NoPadding().WriteWord(",")
		}
	}
	f.WriteString(")").NeedPadding()
}

func (f *formatter) FormatArgument(arg *ast.Argument) {
	f.WriteWord(arg.Name).NoPadding().WriteString(":").NeedPadding()
	f.WriteString(arg.Value.String())
}

func (f *formatter) FormatFragmentDefinitionList(lists ast.FragmentDefinitionList) {
	for _, def := range lists {
		f.FormatFragmentDefinition(def)
	}
}

func (f *formatter) FormatFragmentDefinition(def *ast.FragmentDefinition) {
	f.WriteWord("fragment").WriteWord(def.Name)
	f.FormatVariableDefinitionList(def.VariableDefinition)
	f.WriteWord("on").WriteWord(def.TypeCondition)
	f.FormatDirectiveList(def.Directives)

	if len(def.SelectionSet) != 0 {
		f.FormatSelectionSet(def.SelectionSet)
		f.WriteNewline()
	}
}

func (f *formatter) FormatVariableDefinitionList(lists ast.VariableDefinitionList) {
	if len(lists) == 0 {
		return
	}

	f.WriteString("(")
	for idx, def := range lists {
		f.FormatVariableDefinition(def)

		if idx != len(lists)-1 {
			f.NoPadding().WriteWord(",")
		}
	}
	f.NoPadding().WriteString(")").NeedPadding()
}

func (f *formatter) FormatVariableDefinition(def *ast.VariableDefinition) {
	f.WriteString("$").WriteWord(def.Variable).NoPadding().WriteString(":").NeedPadding()
	f.FormatType(def.Type)

	if def.DefaultValue != nil {
		f.WriteWord("=")
		f.FormatValue(def.DefaultValue)
	}

	// TODO https://github.com/vektah/gqlparser/v2/issues/102
	//   VariableDefinition : Variable : Type DefaultValue? Directives[Const]?
}

func (f *formatter) FormatSelectionSet(sets ast.SelectionSet) {
	if len(sets) == 0 {
		return
	}

	f.WriteString("{").WriteNewline()
	f.IncrementIndent()

	for _, sel := range sets {
		f.FormatSelection(sel)
	}

	f.DecrementIndent()
	f.WriteString("}")
}

func (f *formatter) FormatSelection(selection ast.Selection) {
	switch v := selection.(type) {
	case *ast.Field:
		f.FormatField(v)

	case *ast.FragmentSpread:
		f.FormatFragmentSpread(v)

	case *ast.InlineFragment:
		f.FormatInlineFragment(v)

	default:
		panic(fmt.Errorf("unknown Selection type: %T", selection))
	}

	f.WriteNewline()
}

func (f *formatter) FormatField(field *ast.Field) {
	if field.Alias != "" && field.Alias != field.Name {
		f.WriteWord(field.Alias).NoPadding().WriteString(":").NeedPadding()
	}
	f.WriteWord(field.Name)

	if len(field.Arguments) != 0 {
		f.NoPadding()
		f.FormatArgumentList(field.Arguments)
		f.NeedPadding()
	}

	f.FormatDirectiveList(field.Directives)

	f.FormatSelectionSet(field.SelectionSet)
}

func (f *formatter) FormatFragmentSpread(spread *ast.FragmentSpread) {
	f.WriteWord("...").WriteWord(spread.Name)

	f.FormatDirectiveList(spread.Directives)
}

func (f *formatter) FormatInlineFragment(inline *ast.InlineFragment) {
	f.WriteWord("...")
	if inline.TypeCondition != "" {
		f.WriteWord("on").WriteWord(inline.TypeCondition)
	}

	f.FormatDirectiveList(inline.Directives)

	f.FormatSelectionSet(inline.SelectionSet)
}

func (f *formatter) FormatType(t *ast.Type) {
	f.WriteWord(t.String())
}

func (f *formatter) FormatValue(value *ast.Value) {
	f.WriteString(value.String())
}

func (f *formatter) FormatExtendDefinition(extInfo *extensionInfo) {
	f.WriteWord("extend")

	switch extInfo.Kind {
	case ast.Scalar:
		f.WriteWord("scalar").WriteWord(extInfo.TypeName)
	case ast.Object:
		f.WriteWord("type").WriteWord(extInfo.TypeName)
	case ast.Interface:
		f.WriteWord("interface").WriteWord(extInfo.TypeName)
	case ast.Union:
		f.WriteWord("union").WriteWord(extInfo.TypeName)
	case ast.Enum:
		f.WriteWord("enum").WriteWord(extInfo.TypeName)
	case ast.InputObject:
		f.WriteWord("input").WriteWord(extInfo.TypeName)
	}

	f.FormatFieldList(extInfo.Fields)

	f.WriteNewline()
}

func getSourceName(pos *ast.Position, defaultSource string) string {
	if pos == nil || pos.Src == nil {
		return defaultSource
	}
	return pos.Src.Name
}

func getOrCreateFormatter(name string, formatters map[string]*formatter) *formatter {
	if f, ok := formatters[name]; ok {
		return f
	}
	f := newFormatter(&bytes.Buffer{})
	formatters[name] = f
	return f
}
