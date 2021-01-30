package sql

import (
	"github.com/roneli/fastgql/gql"
	"github.com/spf13/cast"
	"github.com/vektah/gqlparser/v2/ast"
)

type RelationType string

const(
	OneToMany  RelationType = "ONE_TO_MANY"
	OneToOne                = "ONE_TO_ONE"
	ManyToMany              = "MANY_TO_MANY"
)

type relation struct {
	relType           RelationType
	baseTable     	  string
	referenceTable 	  string
	fields     		  []string
	references        []string
	manyToManyTable   string
	manyToManyReferences []string
	manyToManyFields  []string
}

/*
parseRelationDirective parses the sqlRelation directive to connect graphQL Objects with SQL relations, this directive
is also important for creating relational filters.
directive @sqlRelation(relationType: _relationType!, baseTable: String!, refTable: String!, fields: [String!]!,
    references: [String!]!, manyToManyTable: String = "", manyToManyFields: [String] = [], manyToManyReferences: [String] = []) on FIELD_DEFINITION
 */
func parseRelationDirective(d *ast.Directive) relation {
	relType := d.Arguments.ForName("relationType").Value.Raw
	return relation{
		relType:           RelationType(relType),
		fields:  cast.ToStringSlice(gql.GetDirectiveValue(d, "fields")),
		references:  cast.ToStringSlice(gql.GetDirectiveValue(d, "references")),
		baseTable: cast.ToString(gql.GetDirectiveValue(d, "baseTable")),
		referenceTable: cast.ToString(gql.GetDirectiveValue(d, "refTable")),
		manyToManyTable:  cast.ToString(gql.GetDirectiveValue(d, "manyToManyTable")),
		manyToManyFields:  cast.ToStringSlice(gql.GetDirectiveValue(d, "manyToManyFields")),
		manyToManyReferences:  cast.ToStringSlice(gql.GetDirectiveValue(d, "manyToManyReferences")),
	}
}


// getTableName returns the field's type table name in the database, if no directive is defined, type name is presumed
// as teh table's name
func getTableName(schema *ast.Schema, f *ast.FieldDefinition) string {
	objType, ok := schema.Types[f.Type.Name()]
	if !ok {
		return f.Name
	}
	d := objType.Directives.ForName("tableName")
	if d != nil {
		return d.Arguments.ForName("name").Value.Raw
	}
	return f.Name
}
