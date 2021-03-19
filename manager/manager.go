package manager

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/eruca/bisel/net/ws"
	"github.com/eruca/bisel/types"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Manager struct {
	db                 *types.DB
	cacher             types.Cacher
	tablers            []types.Tabler
	handlers           map[string]types.ContextConfig
	configResponseType types.ConfigResponseType
}

// InitSystem 分别启动http,websocket
// 返回可以启动链式操作StartTask
func (manager *Manager) InitSystem(engine *gin.Engine) *Manager {
	engine.POST("/:table/:acid", func(c *gin.Context) {
		table := c.Param("table")
		acid := c.Param("acid")
		router := fmt.Sprintf("%s/%s", table, acid)
		req := types.FromHttpRequest(router, c.Request.Body)
		log.Printf("http request from client: %-v\n", req)

		manager.TakeAction(c.Writer, req, c.Request)
	})

	// 构建读入信息后的处理函数
	processMixHttpRequest := func(httpReq *http.Request) ws.Process {
		return func(send chan<- []byte, msg []byte) {
			req := types.NewRequest(bytes.TrimSpace(msg))
			log.Printf("websocket request from client: %-v\n", req)
			manager.TakeAction(types.NewChanWriter(send), req, httpReq)
		}
	}
	// 连接成功后马上发送的数据
	connected := func(send chan<- []byte) {
		log.Println("Connected now, will send some data to client")
		manager.Connected(send)
	}
	wsHandler := ws.WebsocketHandler(processMixHttpRequest, connected)
	engine.GET("/ws", func(c *gin.Context) {
		wsHandler(c.Writer, c.Request)
	})

	return manager
}

// New ...
func New(gdb *gorm.DB, cacher types.Cacher, configResponseType types.ConfigResponseType,
	tablers ...types.Tabler) *Manager {

	db := &types.DB{Gorm: gdb}
	handlers := make(map[string]types.ContextConfig)
	for _, tabler := range tablers {
		tabler.MustAutoMigrate(db)
		// Register实际上是让tabler自己注册数据到handlers上
		tabler.Register(handlers)
	}
	// 如果未设置，使用默认的
	if configResponseType == nil {
		configResponseType = types.DefaultResponseType
	}

	manager := &Manager{
		db:                 db,
		cacher:             cacher,
		tablers:            tablers,
		handlers:           handlers,
		configResponseType: configResponseType,
	}

	return manager
}

func (manager *Manager) StartTasks(tasks ...types.Task) {
	for _, task := range tasks {
		go task(manager.db, manager.cacher)
	}
}

// TakeAction 可以并发执行
//! Notice: 因为写入都是在初始化阶段，读取可以并发
func (manager *Manager) TakeAction(clientWriter io.Writer, req *types.Request, httpReq *http.Request) (err error) {
	var respReader io.Reader

	if handler, ok := manager.handlers[req.Type]; ok {
		ctx := types.NewContext(manager.db, manager.cacher, req, httpReq, manager.configResponseType)
		handler(ctx)
		ctx.Start()

		if ctx.Responder == nil {
			panic("需要返回一个结果给客户端")
		}
		respReader = types.ResponderToReader(ctx.Responder)
		log.Printf("各个handler结果:\t%v\n", ctx.Results)
	} else {
		resp := types.BuildErrorResposeFromRequest(manager.configResponseType, req, fmt.Errorf("%q router not implemented yet", req.Type))
		respReader = types.ResponderToReader(resp)
	}
	_, err = io.Copy(clientWriter, respReader)
	return
}

// Connected 当连接建立时
func (manager *Manager) Connected(c chan<- []byte) {
	for _, tabler := range manager.tablers {
		if connecter, ok := tabler.(types.Connectter); ok {
			responder := connecter.Connected(manager.db, manager.cacher)
			c <- responder.JSON()
		}
	}
}
