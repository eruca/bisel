package btypes

import (
	"bytes"
)

// Action
//* return: 代表该Action返回的计算的值
type Action func(c *Context) PairStringer

// 对于Context进行配置
// @return represent islogin
type ContextConfig func(*Context) ParamType

// jwtSession: 目的是将jwt的需求构造成一个结构体，发送给客户端就可以里，这个Context也完成使命被回收了
func handlerFunc(tabler Tabler, size int, pt ParamType, jwtSession Defaulter, handlers ...Action) ContextConfig {
	return func(c *Context) ParamType {
		// 重置参数
		tabler = tabler.New()

		// 将客户端发送过来的Payload => ParamsContext
		pc := ParamsContextFromJSON(tabler, size, pt, c.Request.Payload)
		c.config(tabler, &pc, handlers...)

		c.AddActions(func(c *Context) PairStringer {
			var (
				response *Response
				result   Result
				err      error
			)

			// ParamLogin 是构造jwtSession
			// 其他是使用jwtSession
			switch pt {
			case ParamLogin:
				result, err = pc.Do(c.Injected, tabler, jwtSession)
				// 赋值给Context
				c.JwtSession = jwtSession

			case ParamUpsert, ParamDelete, ParamQuery:
				result, err = pc.Do(c.Injected, tabler, c.JwtSession)

			default:
			}
			c.Logger.Debugf("HandlerFunc: %s", pt)

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
		return pt
	}
}

func QueryHandler(tabler Tabler, size int, handlers ...Action) ContextConfig {
	return handlerFunc(tabler, size, ParamQuery, nil, handlers...)
}

func UpsertHandler(tabler Tabler, handlers ...Action) ContextConfig {
	return handlerFunc(tabler, -1, ParamUpsert, nil, handlers...)
}

func DeleteHandler(tabler Tabler, handlers ...Action) ContextConfig {
	return handlerFunc(tabler, -1, ParamDelete, nil, handlers...)
}

func LoginHandler(tabler Tabler, jwtSession Defaulter, handlers ...Action) ContextConfig {
	return handlerFunc(tabler, -1, ParamLogin, jwtSession, handlers...)
}

func LogoutHandler(tabler Tabler, handlers ...Action) ContextConfig {
	return handlerFunc(tabler, -1, ParamLogout, nil, handlers...)
}

func EditOnHandler(tabler Tabler, handlers ...Action) ContextConfig {
	return handlerFunc(tabler, -1, ParamEditOn, nil, handlers...)
}

func EditOffHandler(tabler Tabler, handlers ...Action) ContextConfig {
	return handlerFunc(tabler, -1, ParamEditOff, nil, handlers...)
}
