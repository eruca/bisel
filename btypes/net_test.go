package btypes_test

import (
	"testing"

	"github.com/eruca/bisel/btypes"
	"github.com/stretchr/testify/assert"
)

func TestRawByteAddHash(t *testing.T) {
	rb := btypes.NewRawBytes([]byte("{}"))
	rb.AddHash("xx")

	assert.Equal(t, string("{}"), string(rb.JSON()))
}
