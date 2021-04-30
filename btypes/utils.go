package btypes

import (
	"fmt"
	"io"

	"github.com/eruca/bisel/net/ws"
)

var _ io.Writer = (*ChanWriter)(nil)

// ChanWriter 为了将chan <- []byte与http.ResponseWriter同一实现io.Writer接口
type ChanWriter struct {
	ch chan<- []byte
}

func NewChanWriter(ch chan<- []byte) ChanWriter { return ChanWriter{ch: ch} }

func (c ChanWriter) Write(p []byte) (n int, err error) {
	c.ch <- p
	return len(p), nil
}

type BroadcastChanWriter struct {
	ch       chan ws.BroadcastRequest
	producer chan []byte
}

func NewBroadcastChanWriter(ch chan ws.BroadcastRequest, producer chan []byte) BroadcastChanWriter {
	return BroadcastChanWriter{ch: ch, producer: producer}
}

func (bc BroadcastChanWriter) Write(p []byte) (int, error) {
	if bc.producer == nil {
		panic("需要producer")
	}
	bc.ch <- ws.BroadcastRequest{Data: p, Producer: bc.producer}
	return len(p), nil
}

// Pair 代表Key:Value
type Pair struct {
	Key   string      `json:"key,omitempty"`
	Value interface{} `json:"value,omitempty"`
}

type Pairs []Pair

func (ps *Pairs) Add(key string, value interface{}) {
	*ps = append(*ps, Pair{Key: key, Value: value})
}

// ValueString 直接将string实现fmt.Stringer
type ValueString string

func (vs ValueString) String() string {
	return string(vs)
}

// PairString ...
// type PairString struct {
// 	Key   string      `json:"key,omitempty"`
// 	Value ValueString `json:"value,omitempty"`
// }

// PairStringer 调试时可以比较好控制输出
type PairStringer struct {
	Key   string       `json:"key,omitempty"`
	Value fmt.Stringer `json:"value,omitempty"`
}
