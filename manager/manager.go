package manager

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/eruca/bisel/btypes"
	"github.com/eruca/bisel/logger"
	"github.com/eruca/bisel/middlewares"
	"github.com/eruca/bisel/ws"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Manager struct {
	db                *btypes.DB
	cacher            btypes.Cacher
	tablers           []btypes.Tabler
	handlers          map[string]btypes.ContextConfig
	depends           map[string]map[string]struct{}
	pessimistic_locks map[string]struct{} // 开启了悲观锁
	crt               btypes.ConfigResponseType
	logger            logger.Logger
}

// New Manager
func New(gdb *gorm.DB, cacher btypes.Cacher, logger logger.Logger, jwtAction btypes.Action,
	crt btypes.ConfigResponseType, pessimistic_router string, tablers ...btypes.Tabler) *Manager {

	db := &btypes.DB{Gorm: gdb}
	handlers := make(map[string]btypes.ContextConfig)

	depends := make(map[string]map[string]struct{})
	pessimistic := make(map[string]struct{})

	for _, tabler := range tablers {
		if err := db.Gorm.AutoMigrate(tabler); err != nil {
			panic(err)
		}
		// Register实际上是让tabler自己注册数据到handlers上
		tabler.Register(handlers)

		tableName := tabler.TableName()
		// 注册所有的悲观锁表
		if tabler.PessimisticLock() {
			pessimistic[tableName] = struct{}{}
		}

		// Iterate the tables depend on
		for _, depend := range tabler.Depends() {
			if m, ok := depends[depend]; ok {
				// 如果依赖的表已经存在，也就是该依赖已经有map了
				// 那么需要查看是否tableName在不在m里
				// 如果depend发生改变，会查找所有depends下key为depend里存在的map下所有的key，删除其缓存
				if _, ok = m[tableName]; !ok {
					m[tableName] = struct{}{}
				}
			} else {
				depends[depend] = map[string]struct{}{tableName: {}}
			}
		}
	}

	// 注册悲观锁的handler
	if len(pessimistic) > 0 {
		handlers[pessimistic_router] = middlewares.PessimisticLockHandler(pessimistic, middlewares.TimeElapsed, jwtAction)
	}

	if crt == nil {
		crt = defaultResponseType
	}

	return &Manager{
		db:                db,
		cacher:            cacher,
		tablers:           tablers,
		handlers:          handlers,
		depends:           depends,
		pessimistic_locks: pessimistic,
		crt:               crt,
		logger:            logger,
	}
}

// InitSystem 分别启动http,websocket
// 返回可以启动链式操作StartTask
// @afterConnected => 表示除tabler实现Connectter外，其他想要传送的数据
func (manager *Manager) InitSystem(engine *gin.Engine, afterConnected btypes.Connectter) *Manager {
	engine.POST("/:table/:crud", func(c *gin.Context) {
		table, crud := strings.TrimSpace(c.Param("table")), strings.TrimSpace(c.Param("crud"))
		router := fmt.Sprintf("%s/%s", table, crud)
		// 产生btypes.Request
		req := btypes.FromHttpRequest(router, c.Request.Body)
		manager.logger.Debugf("\nhttp request from client: %-v", req)

		manager.TakeActionHttp(c.Writer, req, c.Request)
	})

	// 构建读入信息后的处理函数
	processMixHttpRequest := func(httpReq *http.Request) (ws.Process, ws.ClearUserID) {
		// 进入该函数，表示一条websocket连接
		// 应该还是在单线程里执行
		return func(client *ws.Client, broadcast chan ws.BroadcastRequest, msg []byte) {
			manager.logger.Warnf("websocket request from client: %s", msg)
			// 产生btypes.Request
			req := btypes.FromJsonMessage(bytes.TrimSpace(msg))
			manager.TakeActionWebsocket(client, broadcast, req, httpReq)
		}, manager.ClearUserID
	}
	// 连接成功后马上发送的数据
	connected := func(send chan<- []byte) {
		manager.logger.Infof("Connected now, will send some data to client")
		manager.Connected(send)
		if afterConnected != nil {
			resp := afterConnected.Push(manager.db, manager.cacher, manager.logger, manager.crt)
			send <- resp.JSON()
		}
	}
	wsHandler := ws.WebsocketHandler(processMixHttpRequest, connected, manager.logger)
	engine.GET("/ws", func(c *gin.Context) {
		wsHandler(c.Writer, c.Request)
	})

	return manager
}

func (manager *Manager) TakeActionWebsocket(client *ws.Client, broadcast chan ws.BroadcastRequest,
	req *btypes.Request, httpReq *http.Request) (err error) {

	if contextConfig, ok := manager.handlers[req.Type]; ok {
		var ctx btypes.Context
		ctx.New(manager.db, manager.cacher, client, httpReq, req,
			manager.depends, manager.pessimistic_locks,
			manager.crt, manager.logger, btypes.WEBSOCKET)

		// 在这里会对paramContext进行初始化, 还没有开始走流程
		contextConfig(&ctx)

		ctx.Logger.Infof("Start Work flow")

		// StartWorkFlow 会启动WorkFlow
		// 并且会走完ctx.Executor,并且会生成一个Responder
		// 走流程之前所有数据都已经准备好了
		ctx.StartWorkFlow()
		if ctx.Responder == nil {
			panic("需要返回一个结果给客户端, 是否在某个middleware中，忘记调用c.Next()了")
		}

		client.Send <- ctx.Responder.JSON()

		if ctx.Responder.Broadcast() {
			ctx.Responder.RemoveUUID()
			ctx.Responder.Silence()
			broadcast <- ws.BroadcastRequest{
				Data:     ctx.Responder.JSON(),
				Producer: client.Send,
			}
		}
		// 打印调用顺序及结果
		ctx.LogResults()
	} else {
		resp := btypes.BuildErrorResposeFromRequest(manager.crt, req,
			fmt.Errorf("%q router not implemented yet", req.Type))
		client.Send <- resp.JSON()
	}

	return
}

// TakeAction 可以并发执行
//! Notice: 因为写入都是在初始化阶段，读取可以并发
func (manager *Manager) TakeActionHttp(clientWriter io.Writer, req *btypes.Request, httpReq *http.Request) (err error) {

	if contextConfig, ok := manager.handlers[req.Type]; ok {
		var ctx btypes.Context
		ctx.New(manager.db, manager.cacher, nil, httpReq, req,
			manager.depends, manager.pessimistic_locks,
			manager.crt, manager.logger, btypes.HTTP)

		// 在这里会对paramContext进行初始化, 还没有开始走流程
		contextConfig(&ctx)

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
		ctx.LogResults()
	} else {
		resp := btypes.BuildErrorResposeFromRequest(manager.crt, req, fmt.Errorf("%q router not implemented yet", req.Type))
		respReader := btypes.ResponderToReader(resp)
		_, err = io.Copy(clientWriter, respReader)
	}

	return
}

func (manager *Manager) ClearUserID(userid uint) {
	key, ok := manager.cacher.Get(userid)
	if !ok {
		// panic("key:userid not exist now, should be")
		manager.logger.Warnf("key:userid not exist now, should be")
		return
	}
	uid, ok := manager.cacher.Get(key)
	if !ok {
		panic(fmt.Sprintf("key: table/id:%q should be exist", key))
	}
	if userid != uid {
		panic(fmt.Sprintf("请求用户%d 与 存储的用户%d 不一致", userid, uid))
	}
	manager.cacher.Remove(key)
	manager.cacher.Remove(userid)
}

// Connected 当连接建立时
func (manager *Manager) Connected(c chan<- []byte) {
	for _, tabler := range manager.tablers {
		if connecter, ok := tabler.(btypes.Connectter); ok {
			responder := connecter.Push(manager.db, manager.cacher, manager.logger, manager.crt)
			c <- responder.JSON()
		}
	}
}

// 否则默认使用responseType作为ConfigResponseType
func defaultResponseType(reqType string, successed bool) string {
	if successed {
		return reqType + "_success"
	}
	return reqType + "_failure"
}
