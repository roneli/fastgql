package sql

import (
	"database/sql/driver"
	"fmt"
)

type wrappedValue struct {
	v interface{}
}

func (w wrappedValue) Value() (driver.Value, error) {
	return w.v, nil
}

func getInputValues(inputValues interface{}) ([]map[string]interface{}, error) {
	switch v := inputValues.(type) {
	case map[string]interface{}:
		return []map[string]interface{}{v}, nil
	case []map[string]interface{}:
		return v, nil
	case []interface{}:
		vals := make([]map[string]interface{}, len(v))
		for i, k := range v {
			vals[i] = k.(map[string]interface{})
		}
		return vals, nil
	default:
		return nil, fmt.Errorf("unexpected value type %T", inputValues)
	}
}
