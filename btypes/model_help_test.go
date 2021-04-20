package btypes_test

import (
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/eruca/bisel/btypes"
	"github.com/stretchr/testify/assert"
)

type Hello struct {
	Good btypes.NullString `json:"good,omitempty"`
	Num  int               `json:"num,omitempty"`
}

func TestNullStringValid(t *testing.T) {
	hello := Hello{Good: btypes.NullString{
		NullString: sql.NullString{
			Valid:  true,
			String: "hello",
		},
	}}

	data, err := json.Marshal(hello)
	assert.NoError(t, err)

	assert.Equal(t, string(data), `{"good":"hello"}`)
}

func TestNullStringInvalid(t *testing.T) {
	hello := Hello{Good: btypes.NullString{
		NullString: sql.NullString{
			Valid:  false,
			String: "hello",
		},
	},
		Num: 1}

	data, err := json.Marshal(hello)
	assert.NoError(t, err)

	assert.Equal(t, string(data), `{"good":null,"num":1}`)
}
