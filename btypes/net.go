package btypes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
)

var (
	_ Responder = (*RawResponse)(nil)
	_ Responder = (*Response)(nil)
)

// ConnectionType 代表连接类型
// 1. http请求 2.websocket请求
type ConnectionType uint8

const (
	HTTP ConnectionType = iota
	WEBSOCKET
)

// ConfigResponseType 让使用者可以定制返回的Type结果
type ConfigResponseType func(string, bool) string

// ResponderToReader 代表将Responder转化为io.Reader
func ResponderToReader(resp Responder) io.Reader {
	return bytes.NewBuffer(resp.JSON())
}

//* Request 从客户端发送过来的请求
type Request struct {
	Type    string          `json:"type,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
	UUID    string          `json:"uuid,omitempty"`
	Token   string          `json:"token,omitempty"`
}

func (req *Request) String() string {
	return fmt.Sprintf(`{"type": %q, "payload": %s,"uuid": %q, "token":%q}`,
		req.Type, req.Payload, req.UUID, req.Token)
}

// NewRequest 将msg解析为*Request
func NewRequest(msg []byte) *Request {
	request := &Request{}
	err := json.Unmarshal(msg, request)
	if err != nil {
		panic(err)
	}
	return request
}

// FromHttpRequest
// http.router => TYPE
// @body => Payload, 如果是query,则可以使用null, 其他不行，所以不能再这里设置
func FromHttpRequest(router string, rder io.ReadCloser) *Request {
	if rder == nil {
		log.Println("rder is nil")
		panic("rder == nil")
	}

	request := &Request{Type: router}
	err := json.NewDecoder(rder).Decode(&request)
	if err != nil && err != io.EOF {
		panic(err)
	}
	rder.Close()
	return request
}

//* Responder 是服务器对客户端的响应的接口
type Responder interface {
	JSON() []byte
	CachePayload() []byte
	Broadcast() bool
}

//* Response 这个是从服务器数据库查询到数据，返回给客户端的响应结果
type Response struct {
	Type      string                 `json:"type,omitempty"`
	Payload   map[string]interface{} `json:"payload"` // payload就是没值也要有{}
	UUID      string                 `json:"uuid,omitempty"`
	broadcast bool
}

// BuildFromRequest 从req，success构建
func BuildFromRequest(responseType ConfigResponseType, req *Request, success, broadcast bool) *Response {
	return &Response{
		Type:      responseType(req.Type, success),
		UUID:      req.UUID,
		broadcast: broadcast,
	}
}

// 如果发生错误就直接生产错误的Response
func BuildErrorResposeFromRequest(responseType ConfigResponseType, req *Request, err error) *Response {
	return &Response{
		Type:      responseType(req.Type, false),
		UUID:      req.UUID,
		Payload:   map[string]interface{}{"err": err.Error()},
		broadcast: false,
	}
}

// Add payload field
func (resp *Response) Add(pairs ...Pair) {
	if resp.Payload == nil {
		resp.Payload = make(map[string]interface{}, len(pairs))
	}
	for _, pair := range pairs {
		resp.Payload[pair.Key] = pair.Value
	}
}

// JSON 实现Responser
func (resp *Response) JSON() []byte {
	data, err := json.Marshal(resp)
	if err != nil {
		panic(err)
	}
	return data
}

func (resp *Response) CachePayload() []byte {
	data, err := json.Marshal(resp.Payload)
	if err != nil {
		panic(err)
	}
	return data
}

// Broadcast 实现Responser
func (resp *Response) Broadcast() bool {
	return resp.broadcast
}

//* RawResponse 就是把所有数据都直接放进去
type RawResponse struct {
	Type    string          `json:"type,omitempty"`
	Payload json.RawMessage `json:"payload"` // payload就是没值也要有{}
	UUID    string          `json:"uuid,omitempty"`
}

func NewRawResponse(crt ConfigResponseType, req *Request, data []byte) *RawResponse {
	return &RawResponse{
		Type:    crt(req.Type, true),
		Payload: data,
		UUID:    req.UUID,
	}
}

// JSON 实现Responser
func (rr RawResponse) JSON() []byte {
	data, err := json.Marshal(rr)
	if err != nil {
		panic(err)
	}
	return data
}

func (rr RawResponse) CachePayload() []byte {
	return rr.Payload
}

// Broadcast ...
func (rr RawResponse) Broadcast() bool { return false }
