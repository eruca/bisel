package btypes_test

import (
	"bufio"
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArraySlice(t *testing.T) {
	hash := md5.New()
	w := bufio.NewWriter(hash)

	buf := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(buf, 0)
	w.Write(buf[:n])

	n = binary.PutUvarint(buf, 1)
	w.Write(buf[:n])
	err := w.Flush()
	assert.NoError(t, err)

	hash2 := md5.New()
	w2 := bufio.NewWriter(hash2)

	buf2 := [binary.MaxVarintLen64]byte{}
	n = binary.PutUvarint(buf2[:], 0)
	w2.Write(buf2[:n])

	n = binary.PutUvarint(buf2[:], 1)
	w2.Write(buf2[:n])
	assert.NoError(t, w2.Flush())

	assert.Equal(t, fmt.Sprintf("%X", hash.Sum(nil)), fmt.Sprintf("%X", hash2.Sum(nil)))
}
