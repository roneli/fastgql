package schema

import (
	"github.com/vektah/gqlparser/v2/ast"
	"log"
	"strings"
)

type (
	Augmenter      func(s *ast.Schema) error
	FieldAugmenter func(s *ast.Schema, obj *ast.Definition, field *ast.FieldDefinition) error
)

func skipAugment(f *ast.FieldDefinition, args ...string) bool {
	if f.Directives.ForName("skipGenerate") != nil || strings.HasPrefix(f.Name, "__") {
		return true
	}
	for _, arg := range args {
		if f.Arguments.ForName(arg) != nil {
			return true
		}
	}
	return false
}

func addRecursive(s *ast.Schema, obj *ast.Definition, fieldStopCase string, augmenter FieldAugmenter, visited ...*ast.Definition) error {
	if hasVisited(obj, visited) {
		return nil
	}
	for _, f := range obj.Fields {
		// avoid recurse and adding to internal objects
		if skipAugment(f, fieldStopCase) || !IsListType(f.Type) {
			log.Printf("skipping field %s@%s for %s\n", f.Name, obj.Name, fieldStopCase)
			continue
		}
		fieldType := s.Types[GetType(f.Type).Name()]
		if !fieldType.IsCompositeType() {
			log.Printf("skipping field %s@%s for %s as its not a compositetype\n", f.Name, obj.Name, fieldStopCase)
			continue
		}
		if err := augmenter(s, obj, f); err != nil {
			log.Printf("error augmenting field %s@%s: %s\n", f.Name, obj.Name, err)
			return err
		}
		return addRecursive(s, fieldType, fieldStopCase, augmenter, append(visited, obj)...)
	}
	return nil
}

func hasVisited(obj *ast.Definition, visited []*ast.Definition) bool {
	for _, v := range visited {
		if v == obj {
			return true
		}
	}
	return false
}
