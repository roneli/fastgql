package sql

import (
	"testing"

	"github.com/doug-martin/goqu/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"

	"github.com/roneli/fastgql/pkg/execution/builders"
)

func TestBuildJsonFieldObject(t *testing.T) {
	tests := []struct {
		name       string
		selections builders.Fields
		wantSQL    string
		wantErr    bool
	}{
		{
			name:       "empty selections returns error",
			selections: builders.Fields{},
			wantSQL:    ``,
			wantErr:    true,
		},
		{
			name: "single scalar field",
			selections: builders.Fields{
				{Field: &ast.Field{Name: "color"}, FieldType: builders.TypeScalar},
			},
			wantSQL: `SELECT jsonb_build_object('color', "test"."attributes"->'color') FROM "test"`,
			wantErr: false,
		},
		{
			name: "multiple scalar fields",
			selections: builders.Fields{
				{Field: &ast.Field{Name: "color"}, FieldType: builders.TypeScalar},
				{Field: &ast.Field{Name: "size"}, FieldType: builders.TypeScalar},
			},
			wantSQL: `SELECT jsonb_build_object('color', "test"."attributes"->'color', 'size', "test"."attributes"->'size') FROM "test"`,
			wantErr: false,
		},
		{
			name: "nested object field",
			selections: builders.Fields{
				{
					Field:     &ast.Field{Name: "details"},
					FieldType: builders.TypeObject,
					Selections: builders.Fields{
						{Field: &ast.Field{Name: "brand"}, FieldType: builders.TypeScalar},
					},
				},
			},
			wantSQL: `SELECT jsonb_build_object('details', jsonb_build_object('brand', "test"."attributes"->'details'->'brand')) FROM "test"`,
			wantErr: false,
		},
		{
			name: "mixed scalar and nested",
			selections: builders.Fields{
				{Field: &ast.Field{Name: "color"}, FieldType: builders.TypeScalar},
				{
					Field:     &ast.Field{Name: "specs"},
					FieldType: builders.TypeObject,
					Selections: builders.Fields{
						{Field: &ast.Field{Name: "weight"}, FieldType: builders.TypeScalar},
					},
				},
			},
			wantSQL: `SELECT jsonb_build_object('color', "test"."attributes"->'color', 'specs', jsonb_build_object('weight', "test"."attributes"->'specs'->'weight')) FROM "test"`,
			wantErr: false,
		},
		{
			name: "deeply nested object",
			selections: builders.Fields{
				{
					Field:     &ast.Field{Name: "outer"},
					FieldType: builders.TypeObject,
					Selections: builders.Fields{
						{
							Field:     &ast.Field{Name: "inner"},
							FieldType: builders.TypeObject,
							Selections: builders.Fields{
								{Field: &ast.Field{Name: "value"}, FieldType: builders.TypeScalar},
							},
						},
					},
				},
			},
			wantSQL: `SELECT jsonb_build_object('outer', jsonb_build_object('inner', jsonb_build_object('value', "test"."attributes"->'outer'->'inner'->'value'))) FROM "test"`,
			wantErr: false,
		},
		{
			name: "nested object with multiple fields",
			selections: builders.Fields{
				{
					Field:     &ast.Field{Name: "details"},
					FieldType: builders.TypeObject,
					Selections: builders.Fields{
						{Field: &ast.Field{Name: "brand"}, FieldType: builders.TypeScalar},
						{Field: &ast.Field{Name: "model"}, FieldType: builders.TypeScalar},
						{Field: &ast.Field{Name: "year"}, FieldType: builders.TypeScalar},
					},
				},
			},
			wantSQL: `SELECT jsonb_build_object('details', jsonb_build_object('brand', "test"."attributes"->'details'->'brand', 'model', "test"."attributes"->'details'->'model', 'year', "test"."attributes"->'details'->'year')) FROM "test"`,
			wantErr: false,
		},
		{
			name: "multiple top-level fields with multiple nested fields",
			selections: builders.Fields{
				{Field: &ast.Field{Name: "color"}, FieldType: builders.TypeScalar},
				{Field: &ast.Field{Name: "size"}, FieldType: builders.TypeScalar},
				{
					Field:     &ast.Field{Name: "specs"},
					FieldType: builders.TypeObject,
					Selections: builders.Fields{
						{Field: &ast.Field{Name: "weight"}, FieldType: builders.TypeScalar},
						{Field: &ast.Field{Name: "height"}, FieldType: builders.TypeScalar},
					},
				},
				{
					Field:     &ast.Field{Name: "details"},
					FieldType: builders.TypeObject,
					Selections: builders.Fields{
						{Field: &ast.Field{Name: "brand"}, FieldType: builders.TypeScalar},
						{Field: &ast.Field{Name: "model"}, FieldType: builders.TypeScalar},
					},
				},
			},
			wantSQL: `SELECT jsonb_build_object('color', "test"."attributes"->'color', 'size', "test"."attributes"->'size', 'specs', jsonb_build_object('weight', "test"."attributes"->'specs'->'weight', 'height', "test"."attributes"->'specs'->'height'), 'details', jsonb_build_object('brand', "test"."attributes"->'details'->'brand', 'model', "test"."attributes"->'details'->'model')) FROM "test"`,
			wantErr: false,
		},
		{
			name: "multiple nested objects at top level",
			selections: builders.Fields{
				{
					Field:     &ast.Field{Name: "specs"},
					FieldType: builders.TypeObject,
					Selections: builders.Fields{
						{Field: &ast.Field{Name: "weight"}, FieldType: builders.TypeScalar},
						{Field: &ast.Field{Name: "height"}, FieldType: builders.TypeScalar},
					},
				},
				{
					Field:     &ast.Field{Name: "details"},
					FieldType: builders.TypeObject,
					Selections: builders.Fields{
						{Field: &ast.Field{Name: "brand"}, FieldType: builders.TypeScalar},
						{Field: &ast.Field{Name: "model"}, FieldType: builders.TypeScalar},
					},
				},
			},
			wantSQL: `SELECT jsonb_build_object('specs', jsonb_build_object('weight', "test"."attributes"->'specs'->'weight', 'height', "test"."attributes"->'specs'->'height'), 'details', jsonb_build_object('brand', "test"."attributes"->'details'->'brand', 'model', "test"."attributes"->'details'->'model')) FROM "test"`,
			wantErr: false,
		},
		{
			name: "deeply nested with multiple fields at each level",
			selections: builders.Fields{
				{
					Field:     &ast.Field{Name: "outer"},
					FieldType: builders.TypeObject,
					Selections: builders.Fields{
						{
							Field:     &ast.Field{Name: "inner"},
							FieldType: builders.TypeObject,
							Selections: builders.Fields{
								{Field: &ast.Field{Name: "value1"}, FieldType: builders.TypeScalar},
								{Field: &ast.Field{Name: "value2"}, FieldType: builders.TypeScalar},
							},
						},
						{Field: &ast.Field{Name: "other"}, FieldType: builders.TypeScalar},
					},
				},
			},
			wantSQL: `SELECT jsonb_build_object('outer', jsonb_build_object('inner', jsonb_build_object('value1', "test"."attributes"->'outer'->'inner'->'value1', 'value2', "test"."attributes"->'outer'->'inner'->'value2'), 'other', "test"."attributes"->'outer'->'other')) FROM "test"`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			col := goqu.T("test").Col("attributes")
			expr, err := BuildJsonFieldObject(col, tt.selections, "postgres")

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			// Build a query to get the SQL
			query := goqu.From("test").Select(expr)
			sqlStr, _, err := query.ToSQL()
			require.NoError(t, err)
			assert.Equal(t, tt.wantSQL, sqlStr)
		})
	}
}
