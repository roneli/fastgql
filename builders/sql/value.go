package sql

import "database/sql/driver"

type wrappedValue struct {
	v interface{}
}

func (w wrappedValue) Value() (driver.Value, error) {
	return w.v, nil
}
