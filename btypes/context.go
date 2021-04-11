package btypes

import (
	"log"
	"net/http"
	"sync"
)

// contextPool 减少Context分配的次数，增加性能
var contextPool sync.Pool = sync.Pool{
	New: func() interface{} {
		return &Context{}
	},
}

type Context struct {
	// 连接类型
	ConnectionType

	*DB
	// 该Tabler 代表需要操作的数据表，比如{journals}/query
	Tabler
	Cacher

	//! todo 登录人员的权限
	JwtSession interface{}

	Executor struct {
		actions []Action
		cursor  int
	}
	// 所有中间件的结果
	Results []PairStringer

	// 这个是http.Request, 是websocket连接的状态
	HttpReq *http.Request
	// *Request 是这条请求信息的Request
	*Request
	Parameters *ParamsContext
	Responder
	// 定制应答类型输出
	ConfigResponseType
}

func NewContext(db *DB, cacher Cacher, req *Request, httpReq *http.Request, cft ConfigResponseType, connType ConnectionType) *Context {
	ctx := contextPool.Get().(*Context)

	ctx.ConnectionType = connType
	ctx.DB = db
	ctx.Cacher = cacher
	ctx.Request = req
	ctx.HttpReq = httpReq
	ctx.ConfigResponseType = cft

	// 初始化其他成员变量
	ctx.Tabler = nil
	ctx.Parameters = nil
	ctx.Executor.actions = nil
	ctx.Executor.cursor = 0
	ctx.Results = nil
	ctx.Responder = nil
	return ctx
}

func (c *Context) config(tabler Tabler, pt *ParamsContext, handlers ...Action) {
	c.Tabler = tabler
	c.Parameters = pt

	c.Executor.actions = make([]Action, 0, len(handlers)+1)
	c.Executor.cursor = 0

	c.Results = make([]PairStringer, 0, len(handlers)+1)
	c.AddActions(handlers...)
}

func (c *Context) AddActions(actions ...Action) {
	c.Executor.actions = append(c.Executor.actions, actions...)
}

func (c *Context) StartWorkFlow() {
	c.exec()
}

func (c *Context) Next() {
	// actions 向前一步
	c.Executor.cursor++
	c.exec()
}

func (c *Context) exec() {
	if c.Executor.cursor < len(c.Executor.actions) {
		result := c.Executor.actions[c.Executor.cursor](c)
		// 保存结果，对应handlers的位置
		c.Results = append(c.Results, result)
	}
}

func (ctx *Context) Finish() {
	ctx.logResults()
	ctx.dispose()
}

func (ctx *Context) dispose() {
	contextPool.Put(ctx)
}

func (c *Context) logResults() {
	log.Printf("'%d'个handler结果:", len(c.Results))
	for i := len(c.Results) - 1; i >= 0; i-- {
		log.Printf("\t%d: %s => %v\n", len(c.Results)-i, c.Results[i].Key, c.Results[i].Value)
	}
}
