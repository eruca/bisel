package ws

import (
	"net/http"
	"time"

	"github.com/eruca/bisel/btypes"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:    4096,
	WriteBufferSize:   4096,
	CheckOrigin:       func(r *http.Request) bool { return true },
	EnableCompression: true,
}

// Client ...
type Client struct {
	conn *websocket.Conn
	send chan []byte
}

func (c *Client) readPump(hub *Hub, fn Process, logger btypes.Logger) {
	defer func() {
		logger.Info("readPump", "client unregister", "conn close")
		hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Info("IsUnexpectedCloseError:", "err", err)
			} else {
				logger.Info("other error:", "err", err)
			}
			break
		}

		fn(c.send, hub.broadcast, message)
	}
}

func (c *Client) writePump(logger btypes.Logger) {
	ticker := time.NewTicker(pingPeriod)

	defer func() {
		ticker.Stop()
		logger.Info("writePump", "conn", "closed")
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, nil)
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				logger.Info("conn nextWriter", "err", err)
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				logger.Info("write close", "err", err)
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				logger.Info("conn.Write Ping Message ", "err", err)
				return
			}
		}
	}
}
