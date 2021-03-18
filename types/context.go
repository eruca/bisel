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
}

func NewContext(db *DB, cacher Cacher, req *Request, httpReq *http.Request) *Context {
	return &Context{DB: db, Cacher: cacher, Request: req, HttpReq: httpReq}
}

func (c *Context) AddActions(actions ...Action) {
	//! +1的目的是为后面加入的action预留位置
	c.Executor.actions = append(c.Executor.actions, actions...)
	c.Results = make([]fmt.Stringer, 0, len(c.Executor.actions)+1)
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
