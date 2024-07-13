package schema

import (
	"log"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"
)

type (
	// Augmenter is a function that modifies the schema, it can be used to add fields, types, etc.
	Augmenter func(s *ast.Schema) error
	// FieldAugmenter is a function that modifies a field in a schema, it can be used to add arguments, directives, etc.
	FieldAugmenter func(s *ast.Schema, obj *ast.Definition, field *ast.FieldDefinition) error
)

// skipAugment checks if the field should be skipped for augmentation, based on the directive skipGenerate
// or if the field name starts with __
func skipAugment(f *ast.FieldDefinition, args ...string) bool {
	if f.Directives.ForName(skipGenerateDirectiveName) != nil || strings.HasPrefix(f.Name, "__") {
		return true
	}
	for _, arg := range args {
		if f.Arguments.ForName(arg) != nil {
			return true
		}
	}
	return false
}

// addRecursive adds the augmenter to all fields of the object and its children
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
		if err := addRecursive(s, fieldType, fieldStopCase, augmenter, append(visited, obj)...); err != nil {
			return err
		}
	}
	return nil
}

// hasVisited checks if the *ast.Definition has been visited already
func hasVisited(obj *ast.Definition, visited []*ast.Definition) bool {
	for _, v := range visited {
		if v == obj {
			return true
		}
	}
	return false
}
