package builders

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
)

func TestFields_HasSelection(t *testing.T) {
	tests := []struct {
		name      string
		fields    Fields
		selection string
		want      bool
	}{
		{
			name:      "exists",
			fields:    Fields{{Field: &ast.Field{Name: "name", Alias: "name"}}},
			selection: "name",
			want:      true,
		},
		{
			name:      "not_exists",
			fields:    Fields{{Field: &ast.Field{Name: "name", Alias: "name"}}},
			selection: "id",
			want:      false,
		},
		{
			name:      "empty_fields",
			fields:    Fields{},
			selection: "name",
			want:      false,
		},
		{
			name:      "multiple_fields_exists",
			fields:    Fields{{Field: &ast.Field{Name: "id", Alias: "id"}}, {Field: &ast.Field{Name: "name", Alias: "name"}}, {Field: &ast.Field{Name: "email", Alias: "email"}}},
			selection: "name",
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fields.HasSelection(tt.selection)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestField_ForName(t *testing.T) {
	tests := []struct {
		name      string
		field     Field
		forName   string
		wantFound bool
	}{
		{
			name: "exists",
			field: Field{
				Selections: Fields{
					{Field: &ast.Field{Name: "id", Alias: "id"}},
					{Field: &ast.Field{Name: "name", Alias: "name"}},
				},
			},
			forName:   "name",
			wantFound: true,
		},
		{
			name: "not_exists",
			field: Field{
				Selections: Fields{
					{Field: &ast.Field{Name: "id", Alias: "id"}},
				},
			},
			forName:   "email",
			wantFound: false,
		},
		{
			name: "empty_selections",
			field: Field{
				Selections: Fields{},
			},
			forName:   "name",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.field.ForName(tt.forName)
			if tt.wantFound {
				require.NoError(t, err)
				assert.Equal(t, tt.forName, got.Name)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestCollectOrdering(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    []OrderField
		wantErr bool
	}{
		{
			name:  "map_single_asc",
			input: map[string]interface{}{"name": "ASC"},
			want:  []OrderField{{Key: "name", Type: OrderingTypesAsc}},
		},
		{
			name:  "map_single_desc",
			input: map[string]interface{}{"id": "DESC"},
			want:  []OrderField{{Key: "id", Type: OrderingTypesDesc}},
		},
		{
			name: "slice_multiple",
			input: []interface{}{
				map[string]interface{}{"name": "ASC"},
				map[string]interface{}{"id": "DESC"},
			},
			want: []OrderField{
				{Key: "name", Type: OrderingTypesAsc},
				{Key: "id", Type: OrderingTypesDesc},
			},
		},
		{
			name:  "null_first",
			input: map[string]interface{}{"name": "ASC_NULL_FIRST"},
			want:  []OrderField{{Key: "name", Type: OrderingTypesAscNull}},
		},
		{
			name:  "desc_null_first",
			input: map[string]interface{}{"name": "DESC_NULL_FIRST"},
			want:  []OrderField{{Key: "name", Type: OrderingTypesDescNull}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CollectOrdering(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.ElementsMatch(t, tt.want, got)
		})
	}
}

func TestGetOperationType(t *testing.T) {
	// Note: GetOperationType requires a full graphql.OperationContext
	// This is tested more thoroughly in integration tests
	// Here we just verify the constants are correct
	assert.Equal(t, OperationType("query"), QueryOperation)
	assert.Equal(t, OperationType("insert"), InsertOperation)
	assert.Equal(t, OperationType("delete"), DeleteOperation)
	assert.Equal(t, OperationType("update"), UpdateOperation)
	assert.Equal(t, OperationType("unknown"), UnknownOperation)
}

func TestWithFieldFilterContext(t *testing.T) {
	tests := []struct {
		name    string
		filters map[string]interface{}
	}{
		{
			name:    "with_filters",
			filters: map[string]interface{}{"id": map[string]interface{}{"eq": 1}},
		},
		{
			name:    "empty_filters",
			filters: map[string]interface{}{},
		},
		{
			name:    "nil_filters",
			filters: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			fc := &FilterFieldContext{Filters: tt.filters}
			newCtx := WithFieldFilterContext(ctx, fc)

			got := GetFieldFilterContext(newCtx)
			require.NotNil(t, got)
			assert.Equal(t, tt.filters, got.Filters)
		})
	}
}

func TestGetFieldFilterContext(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		wantNil bool
	}{
		{
			name:    "exists",
			ctx:     WithFieldFilterContext(context.Background(), &FilterFieldContext{Filters: map[string]interface{}{"id": 1}}),
			wantNil: false,
		},
		{
			name:    "not_exists",
			ctx:     context.Background(),
			wantNil: true,
		},
		{
			name:    "wrong_type",
			ctx:     context.WithValue(context.Background(), filterCtx, "wrong_type"),
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetFieldFilterContext(tt.ctx)
			if tt.wantNil {
				assert.Nil(t, got)
			} else {
				assert.NotNil(t, got)
			}
		})
	}
}

func TestOrderingTypes(t *testing.T) {
	// Verify ordering type constants
	assert.Equal(t, OrderingTypes("ASC"), OrderingTypesAsc)
	assert.Equal(t, OrderingTypes("DESC"), OrderingTypesDesc)
	assert.Equal(t, OrderingTypes("ASC_NULL_FIRST"), OrderingTypesAscNull)
	assert.Equal(t, OrderingTypes("DESC_NULL_FIRST"), OrderingTypesDescNull)
}

func TestInputFieldName(t *testing.T) {
	assert.Equal(t, "inputs", InputFieldName)
}
