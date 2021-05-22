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

// UserRunTimeData 代表登录用户的信息，保存在运行时，主要是保存在Cache里
type UserRuntimeData struct {
	// 在用户表的ID
	UserID uint
	Client *ws.Client // 该用户的send channel

	// 正在编辑的表
	TableName string
	TableID   uint
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

		manager.TakeActionHttp(c.Writer, req, c.Request)
	})

	// 构建读入信息后的处理函数
	processMixHttpRequest := func(httpReq *http.Request) (ws.Process, ws.ClearUserID) {
		// 进入该函数，表示一条websocket连接
		// 应该还是在单线程里执行
		// var websocketDisconneced bool

		return func(client *ws.Client, broadcast chan ws.BroadcastRequest, msg []byte) {
			// 产生btypes.Request
			req := btypes.NewRequest(bytes.TrimSpace(msg))
			manager.logger.Infof("websocket request from client: %-v\n", req)
			manager.TakeActionWebsocket(client, broadcast, req, httpReq)
		}, manager.ClearUserID
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

func (manager *Manager) ClearUserID(userid uint) {
	manager.cacher.Remove(userid)
}

// New Manager
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

	return &Manager{
		db:       db,
		cacher:   cacher,
		tablers:  tablers,
		handlers: handlers,
		crt:      crt,
		config:   config,
		logger:   logger,
	}
}

func (manager *Manager) StartTasks(tasks ...btypes.Task) {
	for _, task := range tasks {
		go task(manager.db, manager.cacher)
	}
}

func (manager *Manager) TakeActionWebsocket(client *ws.Client, broadcast chan ws.BroadcastRequest,
	req *btypes.Request, httpReq *http.Request) (err error) {

	if contextConfig, ok := manager.handlers[req.Type]; ok {
		ctx := btypes.NewContext(manager.db, manager.cacher, req, httpReq, manager.crt,
			manager.logger, manager.config.JWT, btypes.WEBSOCKET)

		// 在这里会对paramContext进行初始化, 还没有开始走流程
		paramType := contextConfig(ctx)

		// StartWorkFlow 会启动WorkFlow
		// 并且会走完ctx.Executor,并且会生成一个Responder
		// 走流程之前所有数据都已经准备好了
		ctx.StartWorkFlow()
		if ctx.Responder == nil {
			panic("需要返回一个结果给客户端, 是否在某个middleware中，忘记调用c.Next()了")
		}

		if ctx.Success && ctx.JwtSession != nil {
			userid := ctx.JwtSession.UserID()

			switch paramType {
			// 登录请求，并且通过验证
			case btypes.ParamLogin:
				client.Userid = userid

				if data, ok := ctx.Cacher.Get(userid); !ok {
					userData := &UserRuntimeData{
						UserID: userid,
						Client: client,
					}
					ctx.Cacher.Set(userid, userData)
				} else {
					if userData, ok1 := data.(*UserRuntimeData); !ok1 {
						ctx.Logger.Errorf("存储的信息不是 *UserRuntimeData")
						panic("存储的信息不是 *UserRuntimeData")
					} else {
						// 如果未退出的情况下，有可能出现该连接已经断开
						if userData.Client.Send != nil {
							userData.Client.Send <- btypes.NewRawResponseText(manager.crt, "users/logout", "", []byte("{}")).JSON()
						}
						// userData.Client.Send = client.Send
						ctx.Cacher.Set(userid, &UserRuntimeData{UserID: userid, Client: client})
					}
				}

			case btypes.ParamLogout:
				// Logout 没有内部的操作
				// 实际上登录患者运行时的数据存储在Cacher里user_id => UserRuntimeData
				// 进行清除工作
				if !ctx.Cacher.Remove(userid) {
					ctx.Logger.Errorf("%d 不在Cache内", userid)
					panic("logout 失败")
				}
			case btypes.ParamEditOn:
				key := fmt.Sprintf("%s/%d", ctx.TableName(), ctx.Tabler.Model().ID)
				if _, ok := ctx.Cacher.Get(key); ok {
					err = btypes.ErrTableIsOnEditting
					break
				} else {
					ctx.Cacher.Set(key, struct{}{})
				}

				loginer_id := ctx.JwtSession.UserID()
				v, ok := ctx.Cacher.Get(loginer_id)
				if !ok {
					ctx.Logger.Errorf("%s:%d 不在Cache内", ctx.TableName(), loginer_id)
					panic("用户不在Cache内")
				}
				urd, ok := v.(*UserRuntimeData)
				if !ok {
					ctx.Logger.Errorf("%s:%d存储的不是*UserRuntimeData", ctx.TableName(), loginer_id)
					panic("存储的数据不是*UserRuntimeData")
				}
				if urd.TableName != "" || urd.TableID > 0 {
					err = btypes.ErrTableIsOnEditting
					break
				}
				urd.TableName = ctx.TableName()
				urd.TableID = ctx.Tabler.Model().ID

			case btypes.ParamEditOff:
				key := fmt.Sprintf("%s/%d", ctx.TableName(), ctx.Tabler.Model().ID)
				if _, ok := ctx.Cacher.Get(key); !ok {
					err = btypes.ErrTableIsOffEditting
					break
				} else {
					ctx.Cacher.Remove(key)
				}

				loginer_id := ctx.JwtSession.UserID()
				v, ok := ctx.Cacher.Get(loginer_id)
				if !ok {
					ctx.Logger.Errorf("%s:%d 不在Cache内", ctx.TableName(), loginer_id)
					panic("用户不在Cache内")
				}
				urd, ok := v.(*UserRuntimeData)
				if !ok {
					ctx.Logger.Errorf("%s:%d存储的不是*UserRuntimeData", ctx.TableName(), loginer_id)
					panic("存储的数据不是*UserRuntimeData")
				}
				if urd.TableName == "" || urd.TableID == 0 {
					err = btypes.ErrTableIsOffEditting
					break
				}
				urd.TableName = ""
				urd.TableID = 0
			}

			if err != nil {
				resp := btypes.BuildErrorResposeFromRequest(manager.crt, req, err)
				client.Send <- resp.JSON()
				return
			}
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

		if paramType == btypes.ParamLogin && ctx.Success {
			manager.Push(client.Send, ctx.JwtSession)
		}

		ctx.Finish()
	} else {
		resp := btypes.BuildErrorResposeFromRequest(manager.crt, req,
			fmt.Errorf("%q router not implemented yet", req.Type))
		client.Send <- resp.JSON()
	}

	return
}

// TakeAction 可以并发执行
//! Notice: 因为写入都是在初始化阶段，读取可以并发
func (manager *Manager) TakeActionHttp(clientWriter io.Writer,
	req *btypes.Request, httpReq *http.Request) (err error) {

	if contextConfig, ok := manager.handlers[req.Type]; ok {
		ctx := btypes.NewContext(manager.db, manager.cacher, req, httpReq, manager.crt,
			manager.logger, manager.config.JWT, btypes.HTTP)

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
			responder.Done()
		}
	}
}

func (manager *Manager) Push(send chan<- []byte, jwtSession btypes.Defaulter) {
	for _, tabler := range manager.tablers {
		if pusher, ok := tabler.(btypes.Pusher); ok && pusher.When() == btypes.Logined {
			if pusher.Auth(jwtSession) {
				responder := pusher.Push(manager.db, manager.cacher, manager.crt, manager.logger)
				send <- responder.JSON()
				responder.Done()
			}
		}
	}
}
