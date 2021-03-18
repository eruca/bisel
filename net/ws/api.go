package ws

import (
	"log"
	"net/http"
	"runtime"
)

// ProcessMixHttpRequest 混入*http.Request
type ProcessMixHttpRequest func(req *http.Request) Process

// Process 是外部函数需要接收websocket的广播、发送、消息, req 代表连接的状态
type Process func(send chan<- []byte, msg []byte)

// Connected 代表如果连接一旦建立，就通过send向客户端发送数据
type Connected func(send chan<- []byte)

// WebsocketHandler 使用方法 获取hub.broadcast
// eg: handler := WebsocketHandler(fn)
// http.HandleFunc("/ws", handler)
//
// ReadProcess 外界需要怎么使用该websocket
// 如果是发生错误直接在源头处理，如果错误是要发送回客户端的也直接序列化为[]byte
// bool表示是否需要广播
// 可以替代cacheFn
//
// 比如连接成功后，客户端发送一个init状态，然后response需要初始化的数据
// WriteClient 直接往broadcast里发送东西，那么会从ReadProcess里读出结果
// 主要是作为websocket发起者时
func WebsocketHandler(process ProcessMixHttpRequest, connected Connected) http.HandlerFunc {
	var hub = newHub()

	// 获取广播接口
	// ServeWs will serve the page request "/ws", and update the http to websocket
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("Connected from", "Host", r.Host, "addr", r.RemoteAddr)

		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			http.Error(w, "服务器错误, 请联系管理员", http.StatusInternalServerError)
			log.Fatalln("upgrage.Upgrade failed", "err", err)
			return
		}

		client := &Client{
			conn: conn,
			send: make(chan []byte, runtime.NumCPU()*2),
		}
		hub.register <- client

		go client.readPump(hub, process(r))
		go client.writePump()

		if connected != nil {
			// 预推送数据, 如果预推送的量超过send的cache量，就会阻塞,
			// 必须在client.writePump启动后再推送
			connected(client.send)
		}
	}
}
