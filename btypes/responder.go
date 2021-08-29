package btypes

import (
	"bytes"
	"encoding/json"
	"io"
)

var (
	_ Responder = (*Response)(nil)
	_ Responder = (*RawResponse)(nil)
)

type Responder interface {
	JSON() []byte
	JSONPayload() []byte
	Broadcast() bool
	RemoveUUID()
	Silence()
}

// ResponderToReader 代表将Responder转化为io.Reader
func ResponderToReader(resp Responder) io.Reader {
	return bytes.NewBuffer(resp.JSON())
}

// *****************************************************************
//* Response 这个是从服务器数据库查询到数据，返回给客户端的响应结果
type Response struct {
	Type      string                 `json:"type,omitempty"`
	Payload   map[string]interface{} `json:"payload"` // payload就是没值也要有{}
	UUID      string                 `json:"uuid,omitempty"`
	broadcast bool
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

func (resp *Response) JSONPayload() []byte {
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

// BuildFromRequest 从req，success构建
func BuildFromRequest(responseType ConfigResponseType, req *Request, success, broadcast bool) *Response {
	resp := &Response{}

	resp.Type = responseType(req.Type, success)
	resp.UUID = req.UUID
	resp.broadcast = broadcast
	return resp
}

// 如果发生错误就直接生产错误的Response
func BuildErrorResposeFromRequest(responseType ConfigResponseType, req *Request, err error) *Response {
	resp := &Response{}

	resp.Type = responseType(req.Type, false)
	resp.UUID = req.UUID
	resp.Payload = map[string]interface{}{"err": err.Error()}
	resp.broadcast = false
	return resp
}

// ***********************************************************************
//* RawResponse 就是把所有数据都直接放进去
type RawResponse struct {
	Type    string          `json:"type,omitempty"`
	Payload json.RawMessage `json:"payload"` // payload就是没值也要有{}
	UUID    string          `json:"uuid,omitempty"`
}

func NewRawResponseText(crt ConfigResponseType, req_type, uuid string, data []byte) *RawResponse {
	rr := &RawResponse{}
	rr.Type = crt(req_type, true)
	rr.Payload = data
	rr.UUID = uuid
	return rr
}

func NewRawResponse(crt ConfigResponseType, req *Request, data []byte) *RawResponse {
	return NewRawResponseText(crt, req.Type, req.UUID, data)
}

// JSON 实现Responser
func (rr *RawResponse) JSON() []byte {
	data, err := json.Marshal(rr)
	if err != nil {
		panic(err)
	}
	return data
}

func (rr *RawResponse) JSONPayload() []byte {
	return rr.Payload
}

// Broadcast ...
func (rr *RawResponse) Broadcast() bool { return false }

func (resp *RawResponse) RemoveUUID() { resp.UUID = "" }

func (resp *RawResponse) Silence() {
	var buf bytes.Buffer

	buf.WriteByte('{')
	buf.WriteString(`"silence":true,`)
	buf.Write(resp.Payload[1:])
	resp.Payload = buf.Bytes()
}
