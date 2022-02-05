package sql

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWrappedValue_Value(t *testing.T) {
	v := wrappedValue{5}
	wv, err := v.Value()
	assert.Nil(t, err)
	assert.Equal(t, 5, wv)
}
