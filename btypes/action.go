package btypes

import (
	"bytes"
)

// Action
//* return: 代表该Action返回的计算的值
type Action func(c *Context) PairStringer

// 对于Context进行配置
type ContextConfig func(*Context)

func handlerFunc(tabler Tabler, pt ParamType, handlers ...Action) ContextConfig {
	return func(c *Context) {
		pc := ParamsContextFromJSON(tabler, pt, c.Request.Payload)
		c.config(tabler, &pc, handlers...)

		c.AddActions(func(c *Context) PairStringer {
			pairs, err := pc.CURD(c.DB, tabler)
			var response *Response
			if err != nil {
				response = BuildErrorResposeFromRequest(c.ConfigResponseType, c.Request, err)
			} else {
				response = BuildFromRequest(c.ConfigResponseType, c.Request, true)
				response.Add(pairs...)
			}
			c.Responder = response

			return c.Parameters.Assemble(bytes.NewBuffer(response.JSON()))
		})
	}
}

func QueryHandler(tabler Tabler, handlers ...Action) ContextConfig {
	return handlerFunc(tabler, ParamQuery, handlers...)
}

func UpsertHandler(tabler Tabler, handlers ...Action) ContextConfig {
	return handlerFunc(tabler, ParamUpsert, handlers...)
}

func DeleteHandler(tabler Tabler, handlers ...Action) ContextConfig {
	return handlerFunc(tabler, ParamDelete, handlers...)
}
