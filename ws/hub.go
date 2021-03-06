package ws

type BroadcastRequest struct {
	Data     []byte
	Producer chan []byte
}

// Hub 代表所有Client的汇集地
type Hub struct {
	clients    map[*Client]struct{}
	broadcast  chan BroadcastRequest
	register   chan *Client
	unregister chan *Client
}

func newHub() *Hub {
	hub := &Hub{
		clients:    make(map[*Client]struct{}),
		broadcast:  make(chan BroadcastRequest),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}

	go hub.run()
	return hub
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = struct{}{}
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
			}
		case req := <-h.broadcast:
			for client := range h.clients {
				if client.Send != req.Producer {
					client.Send <- req.Data
				}
			}
		}
	}
}
