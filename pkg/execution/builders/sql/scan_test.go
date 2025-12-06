package sql

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test types for interface scanning
type Cat struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	Color string `json:"color"`
}

type Dog struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	Breed string `json:"breed"`
}

func TestNewTypeNameScanner(t *testing.T) {
	types := map[string]reflect.Type{
		"Cat": reflect.TypeOf(Cat{}),
		"Dog": reflect.TypeOf(Dog{}),
	}

	t.Run("creates_scanner", func(t *testing.T) {
		scanner := NewTypeNameScanner[any](types, "type")
		assert.NotNil(t, scanner)
	})

	t.Run("lowercase_conversion", func(t *testing.T) {
		scanner := NewTypeNameScanner[any](types, "type")
		// The scanner should lowercase the keys internally
		assert.NotNil(t, scanner.types["cat"])
		assert.NotNil(t, scanner.types["dog"])
	})

	t.Run("preserves_type_key", func(t *testing.T) {
		scanner := NewTypeNameScanner[any](types, "animal_type")
		assert.Equal(t, "animal_type", scanner.typeNameKey)
	})
}

func TestTypeNameScanner_ScanJson(t *testing.T) {
	types := map[string]reflect.Type{
		"cat": reflect.TypeOf(Cat{}),
		"dog": reflect.TypeOf(Dog{}),
	}
	scanner := NewTypeNameScanner[any](types, "type")

	tests := []struct {
		name     string
		json     string
		wantType reflect.Type
		wantErr  bool
	}{
		{
			name:     "valid_cat",
			json:     `{"type":"cat","id":1,"name":"Whiskers","color":"orange"}`,
			wantType: reflect.TypeOf(&Cat{}),
			wantErr:  false,
		},
		{
			name:     "valid_dog",
			json:     `{"type":"dog","id":2,"name":"Rex","breed":"German Shepherd"}`,
			wantType: reflect.TypeOf(&Dog{}),
			wantErr:  false,
		},
		{
			name:     "uppercase_type",
			json:     `{"type":"CAT","id":1,"name":"Fluffy","color":"white"}`,
			wantType: reflect.TypeOf(&Cat{}),
			wantErr:  false,
		},
		{
			name:     "unknown_type",
			json:     `{"type":"bird","id":3,"name":"Tweety"}`,
			wantType: nil,
			wantErr:  true,
		},
		{
			name:     "missing_type_field",
			json:     `{"id":4,"name":"Unknown"}`,
			wantType: nil,
			wantErr:  true,
		},
		{
			name:     "empty_type",
			json:     `{"type":"","id":5,"name":"NoType"}`,
			wantType: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := scanner.ScanJson([]byte(tt.json))
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantType, reflect.TypeOf(got))
		})
	}
}

func TestTypeNameScanner_ScanJson_DataIntegrity(t *testing.T) {
	types := map[string]reflect.Type{
		"cat": reflect.TypeOf(Cat{}),
		"dog": reflect.TypeOf(Dog{}),
	}
	scanner := NewTypeNameScanner[any](types, "type")

	t.Run("cat_data_preserved", func(t *testing.T) {
		json := `{"type":"cat","id":10,"name":"Mittens","color":"black"}`
		got, err := scanner.ScanJson([]byte(json))
		require.NoError(t, err)

		cat, ok := got.(*Cat)
		require.True(t, ok)
		assert.Equal(t, 10, cat.ID)
		assert.Equal(t, "Mittens", cat.Name)
		assert.Equal(t, "cat", cat.Type)
		assert.Equal(t, "black", cat.Color)
	})

	t.Run("dog_data_preserved", func(t *testing.T) {
		json := `{"type":"dog","id":20,"name":"Buddy","breed":"Golden Retriever"}`
		got, err := scanner.ScanJson([]byte(json))
		require.NoError(t, err)

		dog, ok := got.(*Dog)
		require.True(t, ok)
		assert.Equal(t, 20, dog.ID)
		assert.Equal(t, "Buddy", dog.Name)
		assert.Equal(t, "dog", dog.Type)
		assert.Equal(t, "Golden Retriever", dog.Breed)
	})
}

func TestTypeNameScanner_EmptyTypes(t *testing.T) {
	scanner := NewTypeNameScanner[any](map[string]reflect.Type{}, "type")

	_, err := scanner.ScanJson([]byte(`{"type":"anything","id":1}`))
	assert.Error(t, err)
}

func TestTypeNameScanner_DifferentTypeKey(t *testing.T) {
	types := map[string]reflect.Type{
		"cat": reflect.TypeOf(Cat{}),
	}
	scanner := NewTypeNameScanner[any](types, "animal_kind")

	// Should fail because the key is different
	_, err := scanner.ScanJson([]byte(`{"type":"cat","id":1,"name":"Test"}`))
	assert.Error(t, err)

	// Should work with correct key
	json := `{"animal_kind":"cat","id":1,"name":"Test","color":"gray"}`
	got, err := scanner.ScanJson([]byte(json))
	require.NoError(t, err)
	assert.IsType(t, &Cat{}, got)
}
