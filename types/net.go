package types

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

// ConfigResponseType 让使用者可以定制返回的Type结果
type ConfigResponseType func(string, bool) string

// 否则默认使用responseType作为ConfigResponseType
func DefaultResponseType(reqType string, successed bool) string {
	if successed {
		return reqType + "_success"
	}
	return reqType + "_failure"
}

// ResponderToReader 代表将Responder转化为io.Reader
func ResponderToReader(resp Responder) io.Reader {
	return bytes.NewBuffer(resp.JSON())
}

//* Request 从客户端发送过来的请求
type Request struct {
	Type    string          `json:"type,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
	UUID    string          `json:"uuid,omitempty"`
}

func (req *Request) String() string {
	return fmt.Sprintf(`{"type": %q, "payload": %s,"uuid": %q}`, req.Type, req.Payload, req.UUID)
}

func DefaultFetchRequest(tabler Tabler) *Request {
	return &Request{
		Type:    tabler.TableName() + "/fetch",
		Payload: []byte(fmt.Sprintf(`{"size":%d}`, DEFAULT_QUERY_SIZE)),
	}
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
// @body => Payload
func FromHttpRequest(router string, rder io.ReadCloser) *Request {
	request := &Request{Type: router}
	if rder == nil {
		log.Println("rder is nil")
		panic(222)
	}
	err := json.NewDecoder(rder).Decode(&request.Payload)
	if err != nil {
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
type RawBytes struct {
	bytes     []byte
	broadcast bool
}

func NewRawBytes(data []byte) RawBytes { return RawBytes{bytes: data, broadcast: false} }

// JSON 实现Responser
func (rb RawBytes) JSON() []byte {
	return rb.bytes
}

// Broadcast ...
func (rb RawBytes) Broadcast() bool {
	return rb.broadcast
}

func (rb RawBytes) AddHash(string) {}
