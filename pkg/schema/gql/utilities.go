package gql

import (
	"github.com/vektah/gqlparser/v2/ast"
)

func IsListType(a *ast.Type) bool {
	return a.Elem != nil && a.NamedType == ""
}

func IsScalarListType(s *ast.Schema, a *ast.Type) bool {
	if !IsListType(a) {
		return false
	}
	t := GetType(a)
	fieldDef := s.Types[t.Name()]
	// we only support scalar types as aggregate fields
	return fieldDef.IsLeafType()
}

func GetType(a *ast.Type) *ast.Type {
	if a.Elem != nil {
		return GetType(a.Elem)
	}
	return a
}

func GetDirectiveValue(d *ast.Directive, name string) interface{} {
	arg := d.Arguments.ForName(name)
	if arg == nil {
		return nil
	}
	v, _ := arg.Value.Value(nil)
	return v
}
