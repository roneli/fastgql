package gql

import (
	"github.com/vektah/gqlparser/v2/ast"
)

func IsListType(a *ast.Type) bool {
	return a.Elem != nil && a.NamedType == ""
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
