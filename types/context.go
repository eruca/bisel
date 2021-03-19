package types

import (
	"fmt"
	"net/http"
)

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

func NewContext(db *DB, cacher Cacher, req *Request, httpReq *http.Request, cft ConfigResponseType) (ctx *Context) {
	return &Context{DB: db, Cacher: cacher, Request: req, HttpReq: httpReq,ConfigResponseType: cft}
}

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
