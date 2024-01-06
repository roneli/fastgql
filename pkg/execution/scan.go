package execution

import (
	"reflect"
	"strings"

	jsoniter "github.com/json-iterator/go"
)

type InterfaceScanner[T any] interface {
	Scan(data []byte) (T, error)
}

type TypeNameScanner[T any] struct {
	types       map[string]reflect.Type
	defaultType reflect.Type
	typeNameKey string
}

func NewTypeNameScanner[T any](types map[string]reflect.Type, defaultType reflect.Type, typeNameKey string) *TypeNameScanner[T] {
	// lower case all keys
	var t2 = make(map[string]reflect.Type)
	for k, v := range types {
		t2[strings.ToLower(k)] = v
	}
	return &TypeNameScanner[T]{
		types:       t2,
		typeNameKey: typeNameKey,
		defaultType: defaultType,
	}
}

func (t *TypeNameScanner[T]) Scan(data []byte) (T, error) {
	vtn := strings.ToLower(jsoniter.Get(data, t.typeNameKey).ToString())
	valueType, ok := t.types[vtn]
	if !ok {
		var v = reflect.New(t.defaultType).Interface()
		if err := jsoniter.Unmarshal(data, v); err != nil {
			return v.(T), err
		}
		return v.(T), nil
	}
	// dynamically create a new instance of the struct type.
	v := reflect.New(valueType).Interface()
	if err := jsoniter.Unmarshal(data, v); err != nil {
		return v.(T), err
	}
	return v.(T), nil
}
