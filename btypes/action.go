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
func handlerFunc(tabler Tabler, pt ParamType, jwtSession Defaulter, handlers ...Action) ContextConfig {
	return func(c *Context) ParamType {
		// 重置参数
		tabler = tabler.New()

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
			switch pt {
			case ParamLogin:
				result, err = pc.Do(c.Injected, tabler, jwtSession)
				// 赋值给Context
				c.JwtSession = jwtSession
			case ParamLogout:
				// Logout 没有内部的操作
				// 实际上登录患者运行时的数据存储在Cacher里user_id => UserRuntimeData
				// 进行清除工作
				if !c.Cacher.Remove(c.Tabler.Model().ID) {
					c.Logger.Errorf("%d 不在Cache内", c.Tabler.Model().ID)
					panic("logout 失败")
				}
			case ParamEditOn:
				loginer_id := c.JwtSession.UserID()
				v, ok := c.Cacher.Get(loginer_id)
				if !ok {
					c.Logger.Errorf("%s:%d 不在Cache内", c.TableName(), loginer_id)
					panic("用户不在Cache内")
				}
				urd, ok := v.(*UserRuntimeData)
				if !ok {
					c.Logger.Errorf("%s:%d存储的不是*UserRuntimeData", c.TableName(), loginer_id)
					panic("存储的数据不是*UserRuntimeData")
				}
				if urd.TableName != "" || urd.TableID > 0 {
					err = ErrTableIsOnEditting
					break
				}
				urd.TableName = c.TableName()
				urd.TableID = c.Tabler.Model().ID
			case ParamEditOff:
				loginer_id := c.JwtSession.UserID()
				v, ok := c.Cacher.Get(loginer_id)
				if !ok {
					c.Logger.Errorf("%s:%d 不在Cache内", c.TableName(), loginer_id)
					panic("用户不在Cache内")
				}
				urd, ok := v.(*UserRuntimeData)
				if !ok {
					c.Logger.Errorf("%s:%d存储的不是*UserRuntimeData", c.TableName(), loginer_id)
					panic("存储的数据不是*UserRuntimeData")
				}
				if urd.TableName == "" || urd.TableID == 0 {
					err = ErrTableIsOffEditting
					break
				}
				urd.TableName = ""
				urd.TableID = 0
			default:
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
		return pt
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

func LogoutHandler(tabler Tabler, handlers ...Action) ContextConfig {
	return handlerFunc(tabler, ParamLogout, nil, handlers...)
}

func EditOnHandler(tabler Tabler, handlers ...Action) ContextConfig {
	return handlerFunc(tabler, ParamEditOn, nil, handlers...)
}

func EditOffHandler(tabler Tabler, handlers ...Action) ContextConfig {
	return handlerFunc(tabler, ParamEditOff, nil, handlers...)
}
