package btypes

import (
	"bytes"
)

// Action
//* return: 代表该Action返回的计算的值
type Action func(c *Context) PairStringer

// 对于Context进行配置
type ContextConfig func(*Context)

// jwtSession: 目的是将jwt的需求构造成一个结构体，发送给客户端就可以里，这个Context也完成使命被回收了
func handlerFunc(tabler Tabler, pt ParamType, jwtSession Defaulter, handlers ...Action) ContextConfig {
	return func(c *Context) {
		pc := ParamsContextFromJSON(tabler, pt, c.Request.Payload)
		c.config(tabler, &pc, handlers...)

		c.AddActions(func(c *Context) PairStringer {
			var (
				response *Response
				pairs    Pairs
				err      error
			)

			// ParamLogin 是构造jwtSession
			// 其他是使用jwtSession
			if pt == ParamLogin {
				pairs, err = pc.Do(c.DB, tabler, jwtSession)
			} else {
				pairs, err = pc.Do(c.DB, tabler, c.JwtSession)
			}

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
	return handlerFunc(tabler, ParamQuery, nil, handlers...)
}

func UpsertHandler(tabler Tabler, handlers ...Action) ContextConfig {
	return handlerFunc(tabler, ParamUpsert, nil, handlers...)
}

func DeleteHandler(tabler Tabler, handlers ...Action) ContextConfig {
	return handlerFunc(tabler, ParamDelete, nil, handlers...)
}

func LoginHandler(tabler Tabler, jwtSession Defaulter, handlers ...Action) ContextConfig {
	return handlerFunc(tabler, ParamLogin, jwtSession, handlers...)
}
