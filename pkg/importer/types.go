package importer

import (
	schemapkg "github.com/roneli/fastgql/pkg/schema"
	"github.com/vektah/gqlparser/v2/ast"
)

// Schema represents a complete GraphQL schema structure to be generated
type Schema struct {
	// ObjectTypes are the GraphQL object type definitions
	ObjectTypes []*ObjectType
	// QueryFields are fields to add to the Query type
	QueryFields []*QueryField
}

// ObjectType represents a GraphQL object type definition with fastgql directives
type ObjectType struct {
	// Name is the GraphQL type name (PascalCase)
	Name string
	// TableName is the database table name (snake_case)
	TableName string
	// Schema is the database schema name (for PostgreSQL schemas)
	Schema string
	// Dialect is the database dialect (e.g., "postgres")
	Dialect string
	// Fields are the GraphQL field definitions
	Fields []*Field
	// Relations are relationship fields that will have @relation directives
	Relations []*Relation
	// GenerateFilterInput indicates if @generateFilterInput directive should be added
	GenerateFilterInput bool
}

// Field represents a GraphQL field definition
type Field struct {
	// Name is the GraphQL field name (camelCase)
	Name string
	// Type is the GraphQL type name (e.g., "Int", "String", "Boolean")
	Type string
	// IsList indicates if this is a list type [Type]
	IsList bool
	// IsNonNull indicates if this is a non-null type Type!
	IsNonNull bool
	// ColumnName is the database column name (snake_case)
	ColumnName string
	// IsJSON indicates if this field should use @json directive
	IsJSON bool
	// JSONColumnName is the column name for JSON fields
	JSONColumnName string
}

// Relation represents a relationship field with @relation directive
type Relation struct {
	// FieldName is the GraphQL field name for the relation
	FieldName string
	// Type is the relation type (ONE_TO_ONE, ONE_TO_MANY, MANY_TO_MANY)
	Type schemapkg.RelationType
	// Fields are the local fields that form the foreign key
	Fields []string
	// References are the referenced fields in the target table
	References []string
	// TargetType is the GraphQL type name of the related object
	TargetType string
	// ManyToManyTable is the junction table name for MANY_TO_MANY relations
	ManyToManyTable string
	// ManyToManyFields are the fields in the junction table pointing to this table
	ManyToManyFields []string
	// ManyToManyReferences are the fields in the junction table pointing to the target table
	ManyToManyReferences []string
}

// QueryField represents a field to be added to the Query type
type QueryField struct {
	// Name is the field name (typically pluralized table name)
	Name string
	// Type is the GraphQL type name (the ObjectType name)
	Type string
	// IsList indicates if this returns a list [Type]
	IsList bool
	// Generate indicates if @generate directive should be added
	Generate bool
}

// ToASTDefinition converts an ObjectType to a GraphQL AST Definition
func (ot *ObjectType) ToASTDefinition() *ast.Definition {
	def := &ast.Definition{
		Kind: ast.Object,
		Name: ot.Name,
	}

	// Add @table directive
	tableDirective := &ast.Directive{
		Name: schemapkg.TableDirectiveName,
		Arguments: ast.ArgumentList{
			{
				Name: schemapkg.ArgNameTable,
				Value: &ast.Value{
					Kind: ast.StringValue,
					Raw:  `"` + ot.TableName + `"`,
				},
			},
			{
				Name: schemapkg.ArgNameDialect,
				Value: &ast.Value{
					Kind: ast.StringValue,
					Raw:  `"` + ot.Dialect + `"`,
				},
			},
		},
	}

	if ot.Schema != "" {
		tableDirective.Arguments = append(tableDirective.Arguments, &ast.Argument{
			Name: schemapkg.ArgNameSchema,
			Value: &ast.Value{
				Kind: ast.StringValue,
				Raw:  `"` + ot.Schema + `"`,
			},
		})
	}

	def.Directives = []*ast.Directive{tableDirective}

	// Add @generateFilterInput directive if needed
	if ot.GenerateFilterInput {
		def.Directives = append(def.Directives, &ast.Directive{
			Name: schemapkg.FilterInputDirectiveName,
		})
	}

	// Add fields
	for _, f := range ot.Fields {
		def.Fields = append(def.Fields, f.ToASTFieldDefinition())
	}

	// Add relation fields
	for _, r := range ot.Relations {
		def.Fields = append(def.Fields, r.ToASTFieldDefinition())
	}

	return def
}

// ToASTFieldDefinition converts a Field to a GraphQL AST FieldDefinition
func (f *Field) ToASTFieldDefinition() *ast.FieldDefinition {
	fieldDef := &ast.FieldDefinition{
		Name: f.Name,
		Type: f.ToASTType(),
	}

	// Add @json directive if needed
	if f.IsJSON {
		jsonDirective := &ast.Directive{
			Name: schemapkg.JSONDirectiveName,
			Arguments: ast.ArgumentList{
				{
					Name: schemapkg.ArgNameColumn,
					Value: &ast.Value{
						Kind: ast.StringValue,
						Raw:  `"` + f.JSONColumnName + `"`,
					},
				},
			},
		}
		fieldDef.Directives = []*ast.Directive{jsonDirective}
	}

	return fieldDef
}

// ToASTType converts a Field to a GraphQL AST Type
func (f *Field) ToASTType() *ast.Type {
	t := &ast.Type{
		NamedType: f.Type,
	}

	// Apply non-null and list modifiers
	if f.IsList {
		t = &ast.Type{
			Elem: t,
		}
	}

	if f.IsNonNull {
		if t.Elem != nil {
			t.Elem.NonNull = true
		} else {
			t.NonNull = true
		}
	}

	return t
}

// ToASTFieldDefinition converts a Relation to a GraphQL AST FieldDefinition
func (r *Relation) ToASTFieldDefinition() *ast.FieldDefinition {
	// Determine if it's a list type
	isList := r.Type == schemapkg.OneToMany || r.Type == schemapkg.ManyToMany

	fieldType := &ast.Type{
		NamedType: r.TargetType,
	}

	if isList {
		fieldType = &ast.Type{
			Elem: fieldType,
		}
	}

	fieldDef := &ast.FieldDefinition{
		Name: r.FieldName,
		Type: fieldType,
	}

	// Build @relation directive
	relationDirective := &ast.Directive{
		Name: schemapkg.RelationDirectiveName,
		Arguments: ast.ArgumentList{
			{
				Name: schemapkg.ArgNameType,
				Value: &ast.Value{
					Kind: ast.EnumValue,
					Raw:  string(r.Type),
				},
			},
			{
				Name: schemapkg.ArgNameFields,
				Value: &ast.Value{
					Kind: ast.ListValue,
					Raw:  formatStringList(r.Fields),
				},
			},
			{
				Name: schemapkg.ArgNameReferences,
				Value: &ast.Value{
					Kind: ast.ListValue,
					Raw:  formatStringList(r.References),
				},
			},
		},
	}

	// Add many-to-many specific arguments
	if r.Type == schemapkg.ManyToMany {
		if r.ManyToManyTable != "" {
			relationDirective.Arguments = append(relationDirective.Arguments, &ast.Argument{
				Name: schemapkg.ArgNameManyToManyTable,
				Value: &ast.Value{
					Kind: ast.StringValue,
					Raw:  `"` + r.ManyToManyTable + `"`,
				},
			})
		}
		if len(r.ManyToManyFields) > 0 {
			relationDirective.Arguments = append(relationDirective.Arguments, &ast.Argument{
				Name: schemapkg.ArgNameManyToManyFields,
				Value: &ast.Value{
					Kind: ast.ListValue,
					Raw:  formatStringList(r.ManyToManyFields),
				},
			})
		}
		if len(r.ManyToManyReferences) > 0 {
			relationDirective.Arguments = append(relationDirective.Arguments, &ast.Argument{
				Name: schemapkg.ArgNameManyToManyRefs,
				Value: &ast.Value{
					Kind: ast.ListValue,
					Raw:  formatStringList(r.ManyToManyReferences),
				},
			})
		}
	}

	fieldDef.Directives = []*ast.Directive{relationDirective}

	return fieldDef
}

// ToASTFieldDefinition converts a QueryField to a GraphQL AST FieldDefinition
func (qf *QueryField) ToASTFieldDefinition() *ast.FieldDefinition {
	fieldType := &ast.Type{
		NamedType: qf.Type,
	}

	if qf.IsList {
		fieldType = &ast.Type{
			Elem: fieldType,
		}
	}

	fieldDef := &ast.FieldDefinition{
		Name: qf.Name,
		Type: fieldType,
	}

	// Add @generate directive if needed
	if qf.Generate {
		fieldDef.Directives = []*ast.Directive{
			{
				Name: schemapkg.GenerateDirectiveName,
			},
		}
	}

	return fieldDef
}

// formatStringList formats a string slice for GraphQL list value
func formatStringList(strs []string) string {
	// This is a simplified version - in practice, we'd need proper GraphQL formatting
	// For now, return a placeholder that will be properly formatted when generating the schema
	result := "["
	for i, s := range strs {
		if i > 0 {
			result += ", "
		}
		result += `"` + s + `"`
	}
	result += "]"
	return result
}
