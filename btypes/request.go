package btypes

import (
	"encoding/json"
	"io"
)

// *********************************************************************
//* Request 从客户端发送过来的请求
type Request struct {
	Type    string          `json:"type,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
	UUID    string          `json:"uuid,omitempty"`
	Token   string          `json:"token,omitempty"`
}

// FromHttpRequest
// http.router => TYPE
// @body => Payload, 如果是query,则可以使用null, 其他不行，所以不能再这里设置
func FromHttpRequest(router string, rder io.ReadCloser) *Request {
	request := &Request{Type: router}

	err := json.NewDecoder(rder).Decode(&request)
	if err != nil && err != io.EOF {
		panic(err)
	}
	rder.Close()
	return request
}

func FromJsonMessage(msg []byte) *Request {
	req := &Request{}

	err := json.Unmarshal(msg, req)
	if err != nil {
		panic(err)
	}
	return req
}
