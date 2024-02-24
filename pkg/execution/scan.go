package execution

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/jackc/pgx/v5"

	jsoniter "github.com/json-iterator/go"
)

func getTypeName(row pgx.CollectableRow, i int, typeName string) (string, int) {
	if i != -1 && row.FieldDescriptions()[i].Name == typeName {
		return string(row.RawValues()[i]), i
	}
	for i, col := range row.FieldDescriptions() {
		if col.Name == typeName {
			return string(row.RawValues()[i]), i
		}
	}
	return "", -1
}

type InterfaceScanner[T any] interface {
	Scan(data []byte) (T, error)
}

type TypeNameScanner[T any] struct {
	types       map[string]reflect.Type
	typeNameKey string
	lastIndex   int
}

func NewTypeNameScanner[T any](types map[string]reflect.Type, typeNameKey string) *TypeNameScanner[T] {
	// lower case all keys
	var t2 = make(map[string]reflect.Type)
	for k, v := range types {
		t2[strings.ToLower(k)] = v
	}
	return &TypeNameScanner[T]{
		types:       t2,
		typeNameKey: typeNameKey,
		lastIndex:   -1,
	}
}

func (t *TypeNameScanner[T]) ScanRow(row pgx.CollectableRow) (T, error) {
	var (
		typeValue string
		value     T
	)
	typeValue, t.lastIndex = getTypeName(row, t.lastIndex, t.typeNameKey)
	valueType, ok := t.types[typeValue]
	if !ok {
		return value, fmt.Errorf("unknown type %s", typeValue)
	}
	// dynamically create a new instance of the struct type.
	v := reflect.New(valueType).Interface()
	m, err := pgx.RowToMap(row)
	if err != nil {
		return value, err
	}
	if err := mapstructure.Decode(m, v); err != nil {
		return value, err
	}
	return v.(T), nil
}

func (t *TypeNameScanner[T]) ScanJson(data []byte) (T, error) {
	var value T
	vtn := strings.ToLower(jsoniter.Get(data, t.typeNameKey).ToString())
	valueType, ok := t.types[vtn]
	if !ok {
		return value, fmt.Errorf("unknown type %s", vtn)
	}
	// dynamically create a new instance of the struct type.
	v := reflect.New(valueType).Interface()
	if err := jsoniter.Unmarshal(data, v); err != nil {
		return v.(T), err
	}
	return v.(T), nil
}
