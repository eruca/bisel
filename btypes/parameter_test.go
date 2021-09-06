package btypes_test

import (
	"bufio"
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/eruca/bisel/btypes"
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

func TestQueryParameter(t *testing.T) {
	jsonstr := []byte(`{
		"checkjwt": true,
		"check_JWT": true,
		"check_jwt": true,
		"conds":["1","2"],
		"offset":0,
		"size":20,
		"orderby":"updated_at DESC"
	}`)
	query := &btypes.QueryParameter{CheckJWT: true}
	query.FromRawMessage(&btypes.VirtualTable{}, jsonstr)

	assert.True(t, query.JwtCheck())
	assert.Equal(t, query.Conds, []string{"1", "2"})
	assert.Equal(t, query.Offset, uint64(0))
	assert.Equal(t, query.Size, int64(20))
	assert.Equal(t, query.Orderby, "updated_at DESC")
	assert.Equal(t, query.ForceUpdated, false)
	assert.Equal(t, query.String(), "Query")
	assert.Equal(t, query.Status(), btypes.StatusRead)
}

func TestWriterParameter(t *testing.T) {
	tabler := &btypes.VirtualTable{}
	wp := &btypes.WriterParameter{
		CheckJWT:  true,
		ParamType: btypes.ParamInsert,
		Tabler:    tabler,
	}
	jason := []byte(`{
		"check_jwt":false,
		"id":1,
		"version":3
	}`)

	wp.FromRawMessage(wp.Tabler, jason)
	assert.True(t, wp.JwtCheck())
	assert.Equal(t, wp.String(), btypes.ParamInsert.String())
	assert.Panics(t, func() { wp.BuildCacheKey("") })
	assert.Equal(t, wp.Status(), btypes.StatusWrite)
}
