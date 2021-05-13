package ws

import "io"

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
	ch       chan BroadcastRequest
	producer chan []byte
}

func NewBroadcastChanWriter(ch chan BroadcastRequest, producer chan []byte) BroadcastChanWriter {
	return BroadcastChanWriter{ch: ch, producer: producer}
}

func (bc BroadcastChanWriter) Write(p []byte) (int, error) {
	if bc.producer == nil {
		panic("需要producer")
	}
	bc.ch <- BroadcastRequest{Data: p, Producer: bc.producer}
	return len(p), nil
}
