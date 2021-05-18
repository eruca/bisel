package btypes_test

import (
	"testing"

	"github.com/eruca/bisel/btypes"
	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/assert"
)

type Test struct {
	btypes.GormModel `mapstructure:",squash"`
	Hello            string `mapstructure:"hello"`
}

func TestGormMapstructure(t *testing.T) {
	m := map[string]interface{}{
		"id":    1,
		"hello": "hello",
	}

	tst := &Test{}
	err := mapstructure.Decode(m, &tst)
	assert.NoError(t, err)
	assert.Equal(t, tst.ID, uint(1))
	assert.Equal(t, tst.Hello, "hello")
}
