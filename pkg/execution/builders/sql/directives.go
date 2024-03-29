package sql

import (
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/jinzhu/inflection"
	"github.com/vektah/gqlparser/v2/ast"
)

type tableDefinition struct {
	name    string
	schema  string
	objType *ast.Definition
}

func (t tableDefinition) TableExpression() exp.IdentifierExpression {
	tbl := goqu.T(t.name)
	if t.schema != "" {
		return tbl.Schema(t.schema)
	}
	return tbl
}

func (t tableDefinition) String() string {
	if t.schema != "" {
		return fmt.Sprintf(`"%s"."%s"`, t.schema, t.name)
	}
	return fmt.Sprintf(`"%s"`, t.name)
}

// getTableNameFromField returns the field's type tableDefinition name in the database, if no directive is defined, type name is presumed
// as the tableDefinition's name
func getTableNameFromField(schema *ast.Schema, f *ast.FieldDefinition) tableDefinition {
	return getTableName(schema, f.Type.Name(), f.Name)
}

// getTableNameFromField returns the field's type tableDefinition name in the database, if no directive is defined, type name is presumed
// as the tableDefinition's name
func getAggregateTableName(schema *ast.Schema, field *ast.Field) tableDefinition {
	fieldName := strings.Split(field.Name, "Aggregate")[0][1:]
	nonAggField := field.ObjectDefinition.Fields.ForName(fieldName)
	return getTableNameFromField(schema, nonAggField)
}

// getCreateTableName returns the field's type tableDefinition name in the database, if no directive is defined, type name is presumed
// as the tableDefinition's name
func getTableNamePrefix(schema *ast.Schema, prefix string, field *ast.Field) tableDefinition {
	fieldName := strings.Split(field.Name, prefix)[1]
	return getTableName(schema, inflection.Singular(fieldName), fieldName)
}

func getTableName(schema *ast.Schema, typeName, fieldName string) tableDefinition {
	objType, ok := schema.Types[typeName]
	if !ok {
		return tableDefinition{
			name:    fieldName,
			schema:  "",
			objType: nil,
		}
	}
	d := objType.Directives.ForName("table")
	if d == nil {
		return tableDefinition{
			name:    strings.ToLower(objType.Name),
			schema:  "",
			objType: objType,
		}
	}
	name := d.Arguments.ForName("name").Value.Raw
	schemaValue := d.Arguments.ForName("schema")
	if schemaValue == nil {
		return tableDefinition{
			name:    name,
			schema:  "",
			objType: objType,
		}
	}
	return tableDefinition{
		name:    name,
		schema:  schemaValue.Value.Raw,
		objType: objType,
	}
}
