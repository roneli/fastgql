package builders

import (
	"fmt"
	"github.com/vektah/gqlparser/v2/ast"
	"strings"
)


// Builders are called when fields are collected
type QueryBuilder interface {
	OnCollectStart(f *ast.Field, variables map[string]interface{}) error
	OnSingleField(f *ast.Field, variables map[string]interface{}) error
	OnMultiField(f *ast.Field, variables map[string]interface{}) error
	Query() (string, []interface{}, error)
}

type CollectedField struct {
	// Original field definition
	*ast.Field
	Directives []string
	Fields []CollectedField
}

// CollectFields allows for a translator to collect fields and get called by on passed builders
func CollectFields(builder QueryBuilder, f *ast.Field, variables map[string]interface{}) error {

	err := builder.OnCollectStart(f, variables)
	if err != nil {
		return err
	}
	for _, field := range f.SelectionSet {
		switch field := field.(type) {
		case *ast.Field:
			// Auto skip fields that start with with underscore "_"
			if strings.HasPrefix(field.Name, "_") {
				continue
			}
			// Check if collected field should be skipped by directive
			if d := field.Directives.ForName("skip"); d != nil {
				args := d.ArgumentMap(variables)
				if shouldSkip := args["skip"]; shouldSkip != nil {
					continue
				}
			}
			var err error
			if field.SelectionSet != nil {
				err = builder.OnMultiField(field, variables)
			} else {
				err = builder.OnSingleField(field, variables)
			}
			if err != nil {
				return err
			}

		case *ast.FragmentSpread: {
			fmt.Print("fragment spread")
		}
		case *ast.InlineFragment: {
			fmt.Print("inline fragment")
		}
		}
	}
	return nil
}