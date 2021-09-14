package btypes

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/eruca/bisel/logger"
	"github.com/eruca/bisel/ws"
)

type Executor struct {
	actions []Action
	cursor  int
	Results []PairStringer
}

type Context struct {
	// 连接类型
	ConnectionType
	// 数据库
	*DB
	// 缓存
	Cacher
	// Cache depends, readonly in execute
	Depends map[string]map[string]struct{}
	// 开启悲观锁的表
	PessimisticLock map[string]struct{}
	// 日志
	logger.Logger
	// JWT
	JwtSess JwtSession
	// Websocket Client
	WsClient *ws.Client

	Executor

	// Tabler 代表调用方
	Tabler
	// ParamContext代表调用的参数
	// ParamContext
	Parameter

	// 应答的设置
	ConfigResponseType
	// Websocket第一次http请求信息或http请求
	HttpReq *http.Request
	// Websocket或http来的信息，转化为Request
	*Request
	// Request的应答，是一个接口
	Responder
	// 此处应答是否成功
	Success bool
}

func (ctx *Context) Init(db *DB, cacher Cacher, client *ws.Client,
	httpReq *http.Request, req *Request, depends map[string]map[string]struct{},
	pess_lock map[string]struct{}, cft ConfigResponseType,
	logger logger.Logger, connType ConnectionType) {

	ctx.ConnectionType = connType
	ctx.DB = db
	ctx.Cacher = cacher
	ctx.Depends = depends
	ctx.PessimisticLock = pess_lock
	ctx.Request = req
	ctx.HttpReq = httpReq
	ctx.WsClient = client
	ctx.ConfigResponseType = cft
	ctx.Logger = logger

	// 初始化其他成员变量
	ctx.Tabler = nil
	ctx.Parameter = nil
	ctx.Executor.actions = nil
	ctx.Executor.cursor = 0
	ctx.Results = nil
	ctx.Success = false
	ctx.Responder = nil
}

func (c *Context) fill(tabler Tabler, parameter Parameter, handlers ...Action) {
	c.Tabler = tabler
	c.Parameter = parameter

	c.Executor.cursor = 0
	c.Executor.actions = make([]Action, 0, len(handlers)+1)
	c.AddActions(handlers...)

	c.Executor.Results = make([]PairStringer, 0, len(handlers)+1)
}

func (c *Context) AddActions(actions ...Action) {
	c.Executor.actions = append(c.Executor.actions, actions...)
}

func (c *Context) StartWorkFlow() { c.exec() }

func (c *Context) Next() {
	// actions 向前一步
	c.Executor.cursor++
	c.exec()
}

func (c *Context) exec() {
	if c.Executor.cursor < len(c.Executor.actions) {
		result := c.Executor.actions[c.Executor.cursor](c)
		// 保存结果，对应handlers的位置
		c.Executor.Results = append(c.Executor.Results, result)
	}
}

func (c *Context) BuildResponse(result Result, err error) (response *Response) {
	if err != nil {
		response = BuildErrorResposeFromRequest(c.ConfigResponseType, c.Request, err)
	} else {
		response = BuildFromRequest(c.ConfigResponseType, c.Request, true, result.Broadcast)
		response.Add(result.Payloads...)
		c.Success = true
	}
	c.Responder = response
	return
}

func (c *Context) LogResults() {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("'%d' 个handler结果:", len(c.Results)))

	for i := len(c.Results) - 1; i > 0; i-- {
		builder.WriteString(fmt.Sprintf("\n\t\t%d: %s => %v", len(c.Results)-i, c.Results[i].Key, c.Results[i].Value))
	}
	builder.WriteString(fmt.Sprintf("\n\t\t%d: %s => %v\n", len(c.Results), c.Results[0].Key, c.Results[0].Value))
	c.Logger.Infof(builder.String())
}
