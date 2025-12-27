package importer

import (
	"fmt"
	"strings"

	"github.com/iancoleman/strcase"
	schemapkg "github.com/roneli/fastgql/pkg/schema"
	"github.com/vektah/gqlparser/v2/ast"
)

// GenerateSchema converts a Schema structure to a GraphQL AST Source
func GenerateSchema(schema *Schema) (*ast.Source, error) {
	if schema == nil {
		return nil, fmt.Errorf("schema cannot be nil")
	}

	// Build the GraphQL schema string
	var builder strings.Builder

	// Write object type definitions
	for _, objType := range schema.ObjectTypes {
		builder.WriteString(objTypeToGraphQL(objType))
		builder.WriteString("\n\n")
	}

	// Write Query type if there are query fields
	if len(schema.QueryFields) > 0 {
		builder.WriteString(fmt.Sprintf("type %s {\n", schemapkg.GraphQLTypeQuery))
		for _, qf := range schema.QueryFields {
			builder.WriteString("  ")
			builder.WriteString(qfToGraphQL(qf))
			builder.WriteString("\n")
		}
		builder.WriteString("}\n")
	}

	return &ast.Source{
		Name:  "imported_schema",
		Input: builder.String(),
	}, nil
}

// objTypeToGraphQL converts an ObjectType to GraphQL schema string
func objTypeToGraphQL(ot *ObjectType) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("type %s", ot.Name))

	// Add directives
	var directives []string
	tableDirective := fmt.Sprintf(`@%s(name: "%s", dialect: "%s"`, schemapkg.TableDirectiveName, ot.TableName, ot.Dialect)
	if ot.Schema != "" {
		tableDirective += fmt.Sprintf(`, schema: "%s"`, ot.Schema)
	}
	tableDirective += ")"
	directives = append(directives, tableDirective)

	if ot.GenerateFilterInput {
		directives = append(directives, "@"+schemapkg.FilterInputDirectiveName)
	}

	if len(directives) > 0 {
		builder.WriteString(" " + strings.Join(directives, " "))
	}

	builder.WriteString(" {\n")

	// Add fields
	for _, f := range ot.Fields {
		builder.WriteString("  ")
		builder.WriteString(fieldToGraphQL(f))
		builder.WriteString("\n")
	}

	// Add relations
	for _, r := range ot.Relations {
		builder.WriteString("  ")
		builder.WriteString(relationToGraphQL(r))
		builder.WriteString("\n")
	}

	builder.WriteString("}")

	return builder.String()
}

// fieldToGraphQL converts a Field to GraphQL field string
func fieldToGraphQL(f *Field) string {
	var builder strings.Builder

	builder.WriteString(f.Name)
	builder.WriteString(": ")

	// Build type string
	typeStr := f.Type
	if f.IsList {
		typeStr = "[" + typeStr + "]"
	}
	if f.IsNonNull {
		typeStr += "!"
	}

	builder.WriteString(typeStr)

	// Add @json directive if needed
	if f.IsJSON {
		builder.WriteString(fmt.Sprintf(` @%s(column: "%s")`, schemapkg.JSONDirectiveName, f.JSONColumnName))
	}

	return builder.String()
}

// relationToGraphQL converts a Relation to GraphQL field string
func relationToGraphQL(r *Relation) string {
	var builder strings.Builder

	builder.WriteString(r.FieldName)
	builder.WriteString(": ")

	// Build type string
	typeStr := r.TargetType
	if r.Type == schemapkg.OneToMany || r.Type == schemapkg.ManyToMany {
		typeStr = "[" + typeStr + "]"
	}
	builder.WriteString(typeStr)

	// Build @relation directive
	relDirective := fmt.Sprintf(`@%s(type: %s, fields: [%s], references: [%s]`,
		schemapkg.RelationDirectiveName,
		r.Type,
		formatFieldsList(r.Fields),
		formatFieldsList(r.References))

	if r.Type == schemapkg.ManyToMany {
		if r.ManyToManyTable != "" {
			relDirective += fmt.Sprintf(`, %s: "%s"`, schemapkg.ArgNameManyToManyTable, r.ManyToManyTable)
		}
		if len(r.ManyToManyFields) > 0 {
			relDirective += fmt.Sprintf(`, %s: [%s]`, schemapkg.ArgNameManyToManyFields, formatFieldsList(r.ManyToManyFields))
		}
		if len(r.ManyToManyReferences) > 0 {
			relDirective += fmt.Sprintf(`, %s: [%s]`, schemapkg.ArgNameManyToManyRefs, formatFieldsList(r.ManyToManyReferences))
		}
	}

	relDirective += ")"
	builder.WriteString(" " + relDirective)

	return builder.String()
}

// qfToGraphQL converts a QueryField to GraphQL field string
func qfToGraphQL(qf *QueryField) string {
	var builder strings.Builder

	builder.WriteString(qf.Name)
	builder.WriteString(": ")

	typeStr := qf.Type
	if qf.IsList {
		typeStr = "[" + typeStr + "]"
	}
	builder.WriteString(typeStr)

	if qf.Generate {
		builder.WriteString(" @" + schemapkg.GenerateDirectiveName)
	}

	return builder.String()
}

// formatFieldsList formats a slice of field names for GraphQL
func formatFieldsList(fields []string) string {
	quoted := make([]string, len(fields))
	for i, f := range fields {
		quoted[i] = `"` + f + `"`
	}
	return strings.Join(quoted, ", ")
}

// ToPascalCase converts a string to PascalCase (used for type names)
func ToPascalCase(s string) string {
	return strcase.ToCamel(s)
}

// ToCamelCase converts a string to camelCase (used for field names)
func ToCamelCase(s string) string {
	return strcase.ToLowerCamel(s)
}
