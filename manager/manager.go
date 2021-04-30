package manager

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/eruca/bisel/btypes"
	"github.com/eruca/bisel/net/ws"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Manager struct {
	db       *btypes.DB
	cacher   btypes.Cacher
	tablers  []btypes.Tabler
	handlers map[string]btypes.ContextConfig
	Config   ManagerConfig
}

// InitSystem 分别启动http,websocket
// 返回可以启动链式操作StartTask
// @afterConnected => 表示除tabler实现Connectter外，其他想要传送的数据
func (manager *Manager) InitSystem(engine *gin.Engine, afterConnected btypes.Connectter) *Manager {
	engine.POST("/:table/:crud", func(c *gin.Context) {
		table, crud := c.Param("table"), c.Param("crud")
		router := fmt.Sprintf("%s/%s", table, crud)
		req := btypes.FromHttpRequest(router, c.Request.Body)
		log.Printf("http request from client: %-v\n", req)

		manager.TakeAction(c.Writer, nil, req, c.Request, btypes.HTTP)
	})

	// 构建读入信息后的处理函数
	processMixHttpRequest := func(httpReq *http.Request) ws.Process {
		return func(send chan []byte, broadcast chan ws.BroadcastRequest, msg []byte) {
			req := btypes.NewRequest(bytes.TrimSpace(msg))
			log.Printf("websocket request from client: %-v\n", req)
			manager.TakeAction(btypes.NewChanWriter(send), btypes.NewBroadcastChanWriter(broadcast, send), req, httpReq, btypes.WEBSOCKET)
		}
	}
	// 连接成功后马上发送的数据
	connected := func(send chan<- []byte) {
		log.Println("Connected now, will send some data to client")
		manager.Connected(send)
		if afterConnected != nil {
			resp := afterConnected.Connected(manager.db, manager.cacher, manager.Config.ConfigResponseType)
			send <- resp.JSON()
		}
	}
	wsHandler := ws.WebsocketHandler(processMixHttpRequest, connected)
	engine.GET("/ws", func(c *gin.Context) {
		wsHandler(c.Writer, c.Request)
	})

	return manager
}

// New ...
func New(gdb *gorm.DB, cacher btypes.Cacher,
	config ManagerConfig, tablers ...btypes.Tabler) *Manager {

	db := &btypes.DB{Gorm: gdb}
	handlers := make(map[string]btypes.ContextConfig)
	for _, tabler := range tablers {
		tabler.MustAutoMigrate(db)
		// Register实际上是让tabler自己注册数据到handlers上
		tabler.Register(handlers)
	}

	// config初始化使用默认的
	config.init()

	manager := &Manager{
		db:       db,
		cacher:   cacher,
		tablers:  tablers,
		handlers: handlers,
		Config:   config,
	}

	return manager
}

func (manager *Manager) StartTasks(tasks ...btypes.Task) {
	for _, task := range tasks {
		go task(manager.db, manager.cacher)
	}
}

// TakeAction 可以并发执行
//! Notice: 因为写入都是在初始化阶段，读取可以并发
func (manager *Manager) TakeAction(clientWriter, broadcastWriter io.Writer,
	req *btypes.Request, httpReq *http.Request, connType btypes.ConnectionType) (err error) {

	if contextConfig, ok := manager.handlers[req.Type]; ok {
		ctx := btypes.NewContext(manager.db, manager.cacher, req, httpReq, manager.Config.ConfigResponseType, connType)
		// 在这里会对paramContext进行初始化
		contextConfig(ctx)

		// StartWorkFlow 会启动WorkFlow
		// 并且会走完ctx.Executor,并且会生成一个Responder
		ctx.StartWorkFlow()
		if ctx.Responder == nil {
			panic("需要返回一个结果给客户端, 是否在某个middleware中，忘记调用c.Next()了")
		}

		respReader := btypes.ResponderToReader(ctx.Responder)
		_, err = io.Copy(clientWriter, respReader)
		if err != nil {
			panic(err)
		}

		if ctx.Responder.Broadcast() {
			if connType == btypes.HTTP {
				panic(`http 连接 不能广播`)
			}
			ctx.Responder.RemoveUUID()
			ctx.Responder.Silence()
			_, err = io.Copy(broadcastWriter, btypes.ResponderToReader(ctx.Responder))
		}

		ctx.Finish()
	} else {
		resp := btypes.BuildErrorResposeFromRequest(manager.Config.ConfigResponseType, req, fmt.Errorf("%q router not implemented yet", req.Type))
		respReader := btypes.ResponderToReader(resp)
		_, err = io.Copy(clientWriter, respReader)
	}

	return
}

// Connected 当连接建立时
func (manager *Manager) Connected(c chan<- []byte) {
	for _, tabler := range manager.tablers {
		if connecter, ok := tabler.(btypes.Connectter); ok {
			responder := connecter.Connected(manager.db, manager.cacher, manager.Config.ConfigResponseType)
			c <- responder.JSON()
		}
	}
}
