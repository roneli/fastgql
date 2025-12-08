package sql

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWrappedValue_Value(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  interface{}
	}{
		{"int", 5, 5},
		{"string", "hello", "hello"},
		{"float", 3.14, 3.14},
		{"bool_true", true, true},
		{"bool_false", false, false},
		{"nil", nil, nil},
		{"slice", []int{1, 2, 3}, []int{1, 2, 3}},
		{"map", map[string]int{"a": 1}, map[string]int{"a": 1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := wrappedValue{tt.value}
			got, err := v.Value()
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetInputValues(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    []map[string]interface{}
		wantErr bool
	}{
		{
			name:  "single_map",
			input: map[string]interface{}{"name": "test", "id": 1},
			want:  []map[string]interface{}{{"name": "test", "id": 1}},
		},
		{
			name:  "map_slice",
			input: []map[string]interface{}{{"name": "a"}, {"name": "b"}},
			want:  []map[string]interface{}{{"name": "a"}, {"name": "b"}},
		},
		{
			name: "interface_slice",
			input: []interface{}{
				map[string]interface{}{"name": "x"},
				map[string]interface{}{"name": "y"},
			},
			want: []map[string]interface{}{{"name": "x"}, {"name": "y"}},
		},
		{
			name:    "invalid_string",
			input:   "invalid",
			wantErr: true,
		},
		{
			name:    "invalid_int",
			input:   42,
			wantErr: true,
		},
		{
			name:    "invalid_slice_of_strings",
			input:   []string{"a", "b"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getInputValues(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetInputValues_EmptyCases(t *testing.T) {
	t.Run("empty_map", func(t *testing.T) {
		got, err := getInputValues(map[string]interface{}{})
		require.NoError(t, err)
		assert.Len(t, got, 1)
		assert.Empty(t, got[0])
	})

	t.Run("empty_slice", func(t *testing.T) {
		got, err := getInputValues([]map[string]interface{}{})
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("empty_interface_slice", func(t *testing.T) {
		got, err := getInputValues([]interface{}{})
		require.NoError(t, err)
		assert.Empty(t, got)
	})
}
