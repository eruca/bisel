package btypes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
)

var (
	_ Responder = (*RawBytes)(nil)
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
	err := json.NewDecoder(rder).Decode(&request.Payload)
	if err != nil && err != io.EOF {
		panic(err)
	}
	rder.Close()
	return request
}

//* Responder 是服务器对客户端的响应的接口
type Responder interface {
	JSON() []byte
	AddHash(string)
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
func BuildFromRequest(responseType ConfigResponseType, req *Request, success bool) *Response {
	return &Response{
		Type: responseType(req.Type, success),
		UUID: req.UUID,
	}
}

// 如果发生错误就直接生产错误的Response
func BuildErrorResposeFromRequest(responseType ConfigResponseType, req *Request, err error) *Response {
	return &Response{
		Type:    responseType(req.Type, false),
		UUID:    req.UUID,
		Payload: map[string]interface{}{"err": err.Error()},
	}
}

func (resp *Response) AddHash(value string) {
	if resp.Payload == nil {
		resp.Payload = map[string]interface{}{"hash": value}
	} else {
		resp.Payload["hash"] = value
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

// Broadcast 实现Responser
func (resp *Response) Broadcast() bool {
	return resp.broadcast
}

//* RawBytes 就是把所有数据都直接放进去
type RawBytes []byte

func NewRawBytes(data []byte) RawBytes { return data }

// JSON 实现Responser
func (rb RawBytes) JSON() []byte { return rb }

// Broadcast ...
func (rb RawBytes) Broadcast() bool { return false }

func (rb RawBytes) AddHash(string) {}
