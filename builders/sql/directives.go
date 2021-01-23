package sql

import (
	"fastgql/gql"
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
directive @sqlRelation(fields: [String]!, references: [String]!, baseTableName: String, refTableName: String,
    relationTableName: String, relationTableFields: [String]) on FIELD_DEFINITION
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

