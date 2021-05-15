package manager

import (
	"bytes"
	"fmt"
	"io"
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
	crt      btypes.ConfigResponseType
	config   btypes.Config
	logger   btypes.Logger
}

// InitSystem 分别启动http,websocket
// 返回可以启动链式操作StartTask
// @afterConnected => 表示除tabler实现Connectter外，其他想要传送的数据
func (manager *Manager) InitSystem(engine *gin.Engine, afterConnected btypes.Connectter) *Manager {
	engine.POST("/:table/:crud", func(c *gin.Context) {
		table, crud := c.Param("table"), c.Param("crud")
		router := fmt.Sprintf("%s/%s", table, crud)
		// 产生btypes.Request
		req := btypes.FromHttpRequest(router, c.Request.Body)
		manager.logger.Infof("http request from client: %-v\n", req)

		manager.TakeActionHttp(c.Writer, req, c.Request, btypes.HTTP)
	})

	// 构建读入信息后的处理函数
	processMixHttpRequest := func(httpReq *http.Request) ws.Process {
		return func(send chan []byte, broadcast chan ws.BroadcastRequest, msg []byte) {
			// 产生btypes.Request
			req := btypes.NewRequest(bytes.TrimSpace(msg))
			manager.logger.Infof("websocket request from client: %-v\n", req)
			manager.TakeActionWebsocket(send, broadcast, req, httpReq, btypes.WEBSOCKET)
		}
	}
	// 连接成功后马上发送的数据
	connected := func(send chan<- []byte) {
		manager.logger.Infof("Connected now, will send some data to client")
		manager.Connected(send)
		if afterConnected != nil {
			resp := afterConnected.Push(manager.db, manager.cacher, manager.crt, manager.logger)
			send <- resp.JSON()
		}
	}
	wsHandler := ws.WebsocketHandler(processMixHttpRequest, connected, manager.logger)
	engine.GET("/ws", func(c *gin.Context) {
		wsHandler(c.Writer, c.Request)
	})

	return manager
}

// New ...
func New(gdb *gorm.DB, cacher btypes.Cacher, logger btypes.Logger, config btypes.Config, crt btypes.ConfigResponseType, tablers ...btypes.Tabler) *Manager {
	db := &btypes.DB{Gorm: gdb}
	handlers := make(map[string]btypes.ContextConfig)
	for _, tabler := range tablers {
		tabler.MustAutoMigrate(db)
		// Register实际上是让tabler自己注册数据到handlers上
		tabler.Register(handlers)
	}

	if crt == nil {
		crt = defaultResponseType
	}

	manager := &Manager{
		db:       db,
		cacher:   cacher,
		tablers:  tablers,
		handlers: handlers,
		crt:      crt,
		config:   config,
		logger:   logger,
	}

	return manager
}

func (manager *Manager) StartTasks(tasks ...btypes.Task) {
	for _, task := range tasks {
		go task(manager.db, manager.cacher)
	}
}

func (manager *Manager) TakeActionWebsocket(send chan []byte, broadcast chan ws.BroadcastRequest,
	req *btypes.Request, httpReq *http.Request, connType btypes.ConnectionType) (err error) {
	if contextConfig, ok := manager.handlers[req.Type]; ok {
		ctx := btypes.NewContext(manager.db, manager.cacher, req, httpReq, manager.crt,
			manager.logger, manager.config.JWT, connType)

		// 在这里会对paramContext进行初始化, 还没有开始走流程
		isLogin := contextConfig(ctx)

		// StartWorkFlow 会启动WorkFlow
		// 并且会走完ctx.Executor,并且会生成一个Responder
		// 走流程之前所有数据都已经准备好了
		ctx.StartWorkFlow()
		if ctx.Responder == nil {
			panic("需要返回一个结果给客户端, 是否在某个middleware中，忘记调用c.Next()了")
		}

		send <- ctx.Responder.JSON()

		if ctx.Responder.Broadcast() {
			if connType == btypes.HTTP {
				panic(`http 连接 不能广播`)
			}
			ctx.Responder.RemoveUUID()
			ctx.Responder.Silence()
			broadcast <- ws.BroadcastRequest{
				Data:     ctx.Responder.JSON(),
				Producer: send,
			}
		}

		if isLogin && ctx.Success {
			manager.Push(send, ctx.JwtSession)
		}

		ctx.Finish()
	} else {
		resp := btypes.BuildErrorResposeFromRequest(manager.crt, req,
			fmt.Errorf("%q router not implemented yet", req.Type))
		send <- resp.JSON()
	}

	return
}

// TakeAction 可以并发执行
//! Notice: 因为写入都是在初始化阶段，读取可以并发
func (manager *Manager) TakeActionHttp(clientWriter io.Writer,
	req *btypes.Request, httpReq *http.Request, connType btypes.ConnectionType) (err error) {

	if contextConfig, ok := manager.handlers[req.Type]; ok {
		ctx := btypes.NewContext(manager.db, manager.cacher, req, httpReq, manager.crt,
			manager.logger, manager.config.JWT, connType)

		// 在这里会对paramContext进行初始化, 还没有开始走流程
		contextConfig(ctx)

		// StartWorkFlow 会启动WorkFlow
		// 并且会走完ctx.Executor,并且会生成一个Responder
		// 走流程之前所有数据都已经准备好了
		ctx.StartWorkFlow()
		if ctx.Responder == nil {
			panic("需要返回一个结果给客户端, 是否在某个middleware中，忘记调用c.Next()了")
		}

		respReader := btypes.ResponderToReader(ctx.Responder)
		_, err = io.Copy(clientWriter, respReader)
		if err != nil {
			panic(err)
		}

		ctx.Finish()
	} else {
		resp := btypes.BuildErrorResposeFromRequest(manager.crt, req, fmt.Errorf("%q router not implemented yet", req.Type))
		respReader := btypes.ResponderToReader(resp)
		_, err = io.Copy(clientWriter, respReader)
	}

	return
}

// Connected 当连接建立时
func (manager *Manager) Connected(c chan<- []byte) {
	for _, tabler := range manager.tablers {
		if connecter, ok := tabler.(btypes.Connectter); ok {
			responder := connecter.Push(manager.db, manager.cacher, manager.crt, manager.logger)
			c <- responder.JSON()
		}
	}
}

func (manager *Manager) Push(send chan<- []byte, jwtSession btypes.Defaulter) {
	for _, tabler := range manager.tablers {
		if pusher, ok := tabler.(btypes.Pusher); ok && pusher.When() == btypes.Logined {
			if pusher.Auth(jwtSession) {
				responder := pusher.Push(manager.db, manager.cacher, manager.crt, manager.logger)
				send <- responder.JSON()
			}
		}
	}
}
