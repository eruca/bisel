package btypes_test

import (
	"encoding/json"
	"testing"

	"github.com/eruca/bisel/btypes"
	"github.com/stretchr/testify/assert"
)

// 否则默认使用responseType作为ConfigResponseType
func defaultResponseType(reqType string, successed bool) string {
	if successed {
		return reqType + "_success"
	}
	return reqType + "_failure"
}

func TestNewRawResponseText(t *testing.T) {
	rr := btypes.NewRawResponseText(defaultResponseType, "logout", "", nil)

	assert.Equal(t, rr.Payload, json.RawMessage(nil))
	assert.Equal(t, rr.Type, "logout_success")
	assert.Equal(t, rr.UUID, "")

	js := string(rr.JSON())
	assert.Equal(t, js, "{\"type\":\"logout_success\",\"payload\":null}")
}
