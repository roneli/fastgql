package builders

import (
	"fmt"
	"github.com/roneli/fastgql/schema"
	"github.com/spf13/cast"
	"github.com/vektah/gqlparser/v2/ast"
	"strings"
)


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
	if filterArg != nil {
		filter, err := filterArg.Value.Value(variables)
		if err != nil {
			return err
		}
		filterMap, ok := filter.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid filter type")
		}
		if err := BuildFilter(builder, f.Definition, filterMap); err != nil {
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
	switch orderings := value.(type) {
	case map[string]interface{}:
		return builder.OrderBy(buildOrderingHelper(orderings))
	case []interface{}:
		var orderFields []OrderField
		for _, o := range orderings {
			argMap, ok := o.(map[string]interface{})
			if !ok {
				return fmt.Errorf("invalid value type")
			}
			orderFields = append(orderFields, buildOrderingHelper(argMap)...)
		}
		return builder.OrderBy(orderFields)
	default:
		panic(fmt.Sprintf("unknown ordering type %v", orderings))
	}
}

func buildOrderingHelper(argMap map[string]interface{}) []OrderField {
	orderFields := make([]OrderField, len(argMap))
	for k, v := range argMap {
		orderFields = append(orderFields, OrderField{
			Key:  k,
			Type: OrderingTypes(cast.ToString(v)),
		})
	}
	return orderFields
}

func BuildFilter(builder FilterBuilder, field *ast.FieldDefinition, filter map[string]interface{}) error {

	filterInputDef := builder.Config().Schema.Types[fmt.Sprintf("%sFilterInput", field.Type.Name())]
	for k, v := range filter {
		keyType := filterInputDef.Fields.ForName(k).Type
		var err error
		if k == string(schema.LogicalOperatorAND) || k == string(schema.LogicalOperatorOR) {
			vv, ok  := v.([]interface{})
			if !ok {
				return fmt.Errorf("fatal value of logical list exp not list")
			}
			err = builder.Logical(field, schema.LogicalOperator(k), vv)
		} else if k == string(schema.LogicalOperatorNot) {
			err = builder.Logical(field, schema.LogicalOperator(k), []interface{}{v})
		} else if strings.HasSuffix(keyType.Name(), "FilterInput") {
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