package btypes

import (
	"fmt"
	"net/http"
	"sync"
)

var contextPool sync.Pool = sync.Pool{
	New: func() interface{} {
		return &Context{}
	},
}

type Context struct {
	*DB
	Tabler
	Cacher

	Executor struct {
		actions []Action
		cursor  int
	}
	// 所有中间件的结果
	Results []fmt.Stringer

	// 这个是http.Request, 是websocket连接的状态
	HttpReq *http.Request
	// *Request 是这条请求信息的Request
	*Request
	Parameters *ParamsContext
	Responder
	// 定制应答类型输出
	ConfigResponseType
}

func NewContext(db *DB, cacher Cacher, req *Request, httpReq *http.Request, cft ConfigResponseType) *Context {
	ctx := contextPool.Get().(*Context)
	ctx.DB = db
	ctx.Cacher = cacher
	ctx.Request = req
	ctx.HttpReq = httpReq
	ctx.ConfigResponseType = cft

	// 初始化其他成员变量
	ctx.Tabler = nil
	ctx.Executor.actions = nil
	ctx.Executor.cursor = 0
	ctx.Results = nil
	ctx.Parameters = nil
	ctx.Responder = nil
	return ctx
}

func (ctx *Context) Dispose() {
	contextPool.Put(ctx)
}

// func NewContext(db *DB, cacher Cacher, req *Request, httpReq *http.Request, cft ConfigResponseType) (ctx *Context) {
// 	return &Context{DB: db, Cacher: cacher, Request: req, HttpReq: httpReq, ConfigResponseType: cft}
// }

func (c *Context) AddActions(actions ...Action) {
	c.Executor.actions = append(c.Executor.actions, actions...)
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

func (c *Context) Start() {
	c.exec()
}
