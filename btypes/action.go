package btypes

import (
	"bytes"
)

// Action
//* return: 代表该Action返回的计算的值
type Action func(c *Context) PairStringer

// 对于Context进行配置
// @return represent islogin
type ContextConfig func(*Context) bool

// jwtSession: 目的是将jwt的需求构造成一个结构体，发送给客户端就可以里，这个Context也完成使命被回收了
func handlerFunc(tabler Tabler, pt ParamType, jwtSession Defaulter, handlers ...Action) ContextConfig {
	// 重置参数
	tabler = tabler.New()

	return func(c *Context) bool {
		// 将客户端发送过来的Payload => ParamsContext
		pc := ParamsContextFromJSON(tabler, pt, c.Request.Payload)
		c.config(tabler, &pc, handlers...)

		c.AddActions(func(c *Context) PairStringer {
			var (
				response *Response
				result   Result
				err      error
			)

			// ParamLogin 是构造jwtSession
			// 其他是使用jwtSession
			if pt == ParamLogin {
				result, err = pc.Do(c.Injected, tabler, jwtSession)
			} else {
				result, err = pc.Do(c.Injected, tabler, c.JwtSession)
			}

			if err != nil {
				response = BuildErrorResposeFromRequest(c.ConfigResponseType, c.Request, err)
			} else {
				response = BuildFromRequest(c.ConfigResponseType, c.Request, true, result.Broadcast)
				response.Add(result.Payloads...)
				c.Success = true
			}
			c.Responder = response

			return c.Parameters.Assemble(bytes.NewBuffer(response.JSON()))
		})
		return pt == ParamLogin
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
