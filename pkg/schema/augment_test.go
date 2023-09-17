package schema

import (
	_ "embed"
	"fmt"
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/roneli/fastgql/pkg/schema/augmenters"
	"github.com/stretchr/testify/assert"
	"github.com/vektah/gqlparser/v2/ast"
	"testing"
)

//go:embed testdata/add_pagination.graphql
var addPagination string

//go:embed fastgql.graphql
var fastgql string

func Test_AugmentSchema(t *testing.T) {

	tests := []struct {
		name    string
		sources []*ast.Source
		wantErr bool
	}{
		{
			name: "test",
			sources: []*ast.Source{
				{
					Name:  "base.grapqhl",
					Input: addPagination,
				},
			},
		},
	}
	cfg := config.DefaultConfig()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg.Sources = append(tt.sources, &ast.Source{
				Name:    "fastgql.graphql",
				Input:   fastgql,
				BuiltIn: false,
			})
			assert.Nil(t, cfg.LoadSchema())
			sources, err := NewFastGQLPlugin().CreateAugmented(cfg.Schema, augmenters.Pagination{})
			assert.Nil(t, err)
			for _, s := range sources {
				fmt.Println(s.Name)
				fmt.Println(s.Input)
			}
		})
	}

}
