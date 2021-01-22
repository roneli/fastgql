package builders

import (
	"fastgql/schema"
	"fmt"
	"github.com/spf13/cast"
	"github.com/vektah/gqlparser/v2/ast"
	"strings"
)

type CollectedField struct {
	// Original field definition
	*ast.Field
	Directives []string
	Fields []CollectedField
}

func BuildQuery(builder QueryBuilder, f *ast.Field, variables map[string]interface{}) error {
	err := BuildArguments(builder, f, variables)
	if err != nil {
		return err
	}
	return BuildFields(builder, f, variables)
}

// BuildFields allows for a Builder to collect fields and get called
func BuildFields(builder FieldBuilder, f *ast.Field, variables map[string]interface{}) error {

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
				err = builder.OnSelectionField(field, variables)
			} else {
				err = builder.OnSingleField(field, variables)
			}
			if err != nil {
				return err
			}

		case *ast.FragmentSpread: {
			for _, s := range field.Definition.SelectionSet {
				fragmentField, ok := s.(*ast.Field)
				if !ok {
					return fmt.Errorf("expected type of selection field got %s", s)
				}
				if fragmentField.SelectionSet != nil {
					if err := builder.OnSelectionField(fragmentField, variables);  err != nil {
						return err
					}
				} else if err := builder.OnSingleField(fragmentField, variables); err != nil {
					return err
				}

			}

		}
		case *ast.InlineFragment: {
			fmt.Print("inline fragment")
		}
		}
	}
	return nil
}

func BuildArguments(builder ArgumentsBuilder, f *ast.Field, variables map[string]interface{}) error {
	limitArg := f.Arguments.ForName("limit")
	if limitArg != nil {
		limit, err := limitArg.Value.Value(variables)
		if err != nil {
			return err
		}
		if err := builder.Limit(cast.ToUint(limit)); err != nil {
			return err
		}
	}
	offsetArg := f.Arguments.ForName("offset")
	if offsetArg != nil {
		offset, err := offsetArg.Value.Value(variables)
		if err != nil {
			return err
		}
		if err := builder.Offset(cast.ToUint(offset)); err != nil {
			return err
		}
	}

	orderingArg := f.Arguments.ForName("orderBy")
	if orderingArg != nil {
		if err := BuildOrdering(builder, orderingArg, variables); err != nil {
			return err
		}
	}

	filterArg := f.Arguments.ForName("filter")
	if offsetArg != nil {
		filter, err := filterArg.Value.Value(variables)
		if err != nil {
			return err
		}
		filterMap, ok := filter.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid filter type")
		}
		if err := BuildFilter(builder, f, filterMap); err != nil {
			return err
		}
	}
	return nil
}

func BuildOrdering(builder OrderingBuilder, arg *ast.Argument, variables map[string]interface{}) error {
	value, err := arg.Value.Value(variables)
	if err != nil {
		return err
	}
	argMap, ok := value.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid value type")
	}

	orderFields := make([]OrderField, len(argMap))
	for k, v := range argMap {
		orderFields = append(orderFields, OrderField{
			Key:  k,
			Type: OrderingTypes(cast.ToString(v)),
		})
	}
	return builder.OrderBy(orderFields)
}

func BuildFilter(builder FilterBuilder, field *ast.Field, filter map[string]interface{}) error {
	for k, v := range filter {
		var err error
		if k == string(schema.LogicalOperatorAND) || k == string(schema.LogicalOperatorOR) {
			vv, ok  := v.([]interface{})
			if !ok {
				return fmt.Errorf("fatal value of logical list exp not list")
			}
			err = builder.Logical(field, schema.LogicalOperator(k), vv)

		} else if strings.HasSuffix(k, "BoolExp") {
			kv, ok  := v.(map[string]interface{})
			if !ok {
				return fmt.Errorf("fatal value of bool exp not map")
			}
			err = builder.Filter(field, k, kv)
		} else {
			opMap, ok  := v.(map[string]interface{})
			if !ok {
				return fmt.Errorf("fatal value of key not map")
			}
			for op, value := range opMap {
				err = builder.Operation(k, op ,value)
			}
		}
		if err != nil {
			return err
		}
	}
	return nil
}