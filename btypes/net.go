package btypes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sync"
)

var (
	_ Responder = (*RawResponse)(nil)
	_ Responder = (*Response)(nil)

	// Pools
	requestPool = &sync.Pool{
		New: func() interface{} {
			return &Request{}
		},
	}
	responsePool = &sync.Pool{
		New: func() interface{} {
			return &Response{}
		},
	}
	rawResponsePool = &sync.Pool{
		New: func() interface{} {
			return &RawResponse{}
		},
	}
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

// *********************************************************************
//* Request 从客户端发送过来的请求
type Request struct {
	Type    string          `json:"type,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
	UUID    string          `json:"uuid,omitempty"`
	Token   string          `json:"token,omitempty"`
}

func newRequest() *Request {
	req := requestPool.Get().(*Request)
	req.Type = ""
	req.Payload = nil
	req.UUID = ""
	req.Token = ""

	return req
}

func (req *Request) Done() {
	requestPool.Put(req)
}

func (req *Request) String() string {
	return fmt.Sprintf(`{"type": %q, "payload": %s,"uuid": %q, "token":%q}`,
		req.Type, req.Payload, req.UUID, req.Token)
}

// NewRequest 将msg解析为*Request
func NewRequest(msg []byte) *Request {
	request := newRequest()

	err := json.Unmarshal(msg, request)
	if err != nil {
		panic(err)
	}
	return request
}

func NewRequestWithType(t string) *Request {
	request := newRequest()
	request.Type = t
	return request
}

// FromHttpRequest
// http.router => TYPE
// @body => Payload, 如果是query,则可以使用null, 其他不行，所以不能再这里设置
func FromHttpRequest(router string, rder io.ReadCloser) *Request {
	request := NewRequestWithType(router)
	err := json.NewDecoder(rder).Decode(&request)
	if err != nil && err != io.EOF {
		panic(err)
	}
	rder.Close()
	return request
}

// *******************************************************************
//* Responder 是服务器对客户端的响应的接口
type Responder interface {
	JSON() []byte
	CachePayload() []byte
	Broadcast() bool
	RemoveUUID()
	Silence()
	Done()
}

// *****************************************************************
//* Response 这个是从服务器数据库查询到数据，返回给客户端的响应结果
type Response struct {
	Type      string                 `json:"type,omitempty"`
	Payload   map[string]interface{} `json:"payload"` // payload就是没值也要有{}
	UUID      string                 `json:"uuid,omitempty"`
	broadcast bool
}

func newResponse() *Response {
	resp := responsePool.Get().(*Response)
	resp.Type = ""
	resp.Payload = nil
	resp.UUID = ""
	resp.broadcast = false
	return resp
}

// BuildFromRequest 从req，success构建
func BuildFromRequest(responseType ConfigResponseType, req *Request, success, broadcast bool) *Response {
	resp := newResponse()

	resp.Type = responseType(req.Type, success)
	resp.UUID = req.UUID
	resp.broadcast = broadcast
	return resp
}

// 如果发生错误就直接生产错误的Response
func BuildErrorResposeFromRequest(responseType ConfigResponseType, req *Request, err error) *Response {
	resp := newResponse()

	resp.Type = responseType(req.Type, false)
	resp.UUID = req.UUID
	resp.Payload = map[string]interface{}{"err": err.Error()}
	resp.broadcast = false
	return resp
}

func (resp *Response) Done() {
	responsePool.Put(resp)
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

func (resp *Response) RemoveUUID() {
	resp.UUID = ""
}

func (resp *Response) Silence() {
	resp.Add(Pair{Key: "silence", Value: true})
}

// ***********************************************************************
//* RawResponse 就是把所有数据都直接放进去
type RawResponse struct {
	Type    string          `json:"type,omitempty"`
	Payload json.RawMessage `json:"payload"` // payload就是没值也要有{}
	UUID    string          `json:"uuid,omitempty"`
}

func newRawResponse() *RawResponse {
	rr := rawResponsePool.Get().(*RawResponse)
	rr.Type = ""
	rr.Payload = nil
	rr.UUID = ""
	return rr
}

func NewRawResponse(crt ConfigResponseType, req *Request, data []byte) *RawResponse {
	rr := newRawResponse()
	rr.Type = crt(req.Type, true)
	rr.Payload = data
	rr.UUID = req.UUID
	return rr
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

func (resp *RawResponse) RemoveUUID() { resp.UUID = "" }

func (resp *RawResponse) Silence() {
	var buf bytes.Buffer

	buf.WriteByte('{')
	buf.WriteString(`"silence":true,`)
	buf.Write(resp.Payload[1:])
	resp.Payload = buf.Bytes()
}

func (resp *RawResponse) Done() {
	rawResponsePool.Put(resp)
}
