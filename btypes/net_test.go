package btypes_test

import (
	"fmt"
	"testing"

	"github.com/eruca/bisel/btypes"
	"github.com/stretchr/testify/assert"
)

func TestRawByteAddHash(t *testing.T) {
	rb := btypes.NewRawBytes([]byte("{}"))
	rb.AddHash("xx")

	assert.Equal(t, string([]byte(fmt.Sprintf("{%q:%q,}", "hash", "xx"))), string(rb.JSON()))
}
