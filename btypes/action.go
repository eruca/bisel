package btypes

import (
	"bytes"
	"fmt"
)

// Action
//* return: 代表该Action返回的计算的值
type Action func(c *Context) fmt.Stringer

// 对于Context进行配置
type ContextConfig func(*Context)

func HandlerFunc(tabler Tabler, pt ParamType, handlers ...Action) ContextConfig {
	return func(c *Context) {
		c.Tabler = tabler
		pc := ParamsContextFromJSON(tabler, pt, c.Request.Payload)
		c.Parameters = &pc

		c.Executor.actions = make([]Action, 0, len(handlers)+1)
		c.Results = make([]fmt.Stringer, 0, len(handlers)+1)
		c.AddActions(handlers...)
		c.AddActions(func(c *Context) fmt.Stringer {
			pairs, err := pc.CURD(c.DB, tabler)
			var response *Response
			if err != nil {
				response = BuildErrorResposeFromRequest(c.ConfigResponseType, c.Request, err)
			} else {
				if pc.ParamType == ParamDelete || pc.ParamType == ParamUpsert {
					c.Cacher.Clear(tabler.TableName())
				}

				response = BuildFromRequest(c.ConfigResponseType, c.Request, true)
				response.Add(pairs...)
			}
			c.Responder = response

			return bytes.NewBuffer(response.JSON())
		})
	}
}
