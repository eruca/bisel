package btypes_test

import (
	"testing"

	"github.com/eruca/bisel/btypes"
	"github.com/stretchr/testify/assert"
)

func TestPairs_Push(t *testing.T) {
	pair := btypes.Pair{Key: "1", Value: 1}
	pairs := btypes.Pairs{}

	pairs.Push(pair)
	assert.Equal(t, len(pairs), 1)
	assert.Equal(t, pairs[0], pair)
}
