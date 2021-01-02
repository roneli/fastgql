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
	baseTableName     string
	relationTableName string
	baseTableKeys     []string
	relationTableKeys []string
}


func parseRelationDirective(d *ast.Directive) relation {
	relType := d.Arguments.ForName("relationType").Value.Raw
	return relation{
		relType:           RelationType(relType),
		baseTableName:     cast.ToString(gql.GetDirectiveValue(d, "baseTableName")),
		relationTableName: cast.ToString(gql.GetDirectiveValue(d, "relTableName")),
		baseTableKeys:     cast.ToStringSlice(gql.GetDirectiveValue(d, "baseTableKeys")),
		relationTableKeys: cast.ToStringSlice(gql.GetDirectiveValue(d, "relTableKeys")),
	}
}
