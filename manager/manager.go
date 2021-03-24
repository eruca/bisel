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
		table := c.Param("table")
		crud := c.Param("crud")
		router := fmt.Sprintf("%s/%s", table, crud)
		req := btypes.FromHttpRequest(router, c.Request.Body)
		log.Printf("http request from client: %-v\n", req)

		manager.TakeAction(c.Writer, req, c.Request)
	})

	// 构建读入信息后的处理函数
	processMixHttpRequest := func(httpReq *http.Request) ws.Process {
		return func(send chan<- []byte, msg []byte) {
			req := btypes.NewRequest(bytes.TrimSpace(msg))
			log.Printf("websocket request from client: %-v\n", req)
			manager.TakeAction(btypes.NewChanWriter(send), req, httpReq)
		}
	}
	// 连接成功后马上发送的数据
	connected := func(send chan<- []byte) {
		log.Println("Connected now, will send some data to client")
		manager.Connected(send)
		if afterConnected != nil {
			resp := afterConnected.Connected(manager.db, manager.cacher)
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
func (manager *Manager) TakeAction(clientWriter io.Writer, req *btypes.Request, httpReq *http.Request) (err error) {
	var respReader io.Reader

	if handler, ok := manager.handlers[req.Type]; ok {
		ctx := btypes.NewContext(manager.db, manager.cacher, req, httpReq, manager.Config.ConfigResponseType)
		// 在这里会对paramContext进行初始化
		handler(ctx)
		ctx.Start()

		if ctx.Responder == nil {
			panic("需要返回一个结果给客户端")
		}
		respReader = btypes.ResponderToReader(ctx.Responder)
		ctx.LogResults()
		ctx.Dispose()
	} else {
		resp := btypes.BuildErrorResposeFromRequest(manager.Config.ConfigResponseType, req, fmt.Errorf("%q router not implemented yet", req.Type))
		respReader = btypes.ResponderToReader(resp)
	}
	_, err = io.Copy(clientWriter, respReader)
	return
}

// Connected 当连接建立时
func (manager *Manager) Connected(c chan<- []byte) {
	for _, tabler := range manager.tablers {
		if connecter, ok := tabler.(btypes.Connectter); ok {
			responder := connecter.Connected(manager.db, manager.cacher)
			c <- responder.JSON()
		}
	}
}
