package types

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
		pc.Init()
		c.Parameters = &pc

		c.Executor.actions = make([]Action, 0, len(handlers)+1)
		c.Results = make([]fmt.Stringer, 0, len(handlers)+1)
		c.AddActions(handlers...)
		c.AddActions(func(c *Context) fmt.Stringer {
			pairs, err := pc.CURD(c.DB, tabler)
			var response *Response
			if err != nil {
				response = BuildErrorResposeFromRequest(c.Request, err)
			} else {
				if pc.ParamType == ParamDelete || pc.ParamType == ParamUpsert {
					c.Cacher.Clear(tabler.TableName())
				}

				response = BuildFromRequest(c.Request, true)
				response.Add(pairs...)
			}
			c.Responder = response

			return bytes.NewBuffer(response.JSON())
		})
	}
}

// BuildFetchHandleFunc 创建Fetch通用的函数
// func BuildFetchHandleFunc(tabler Tabler, handlers ...Action) ContextConfig {
// 	return func(c *Context) {
// 		c.Tabler = tabler
// 		params := QueryParams{}
// 		err := json.Unmarshal(c.Request.Payload, &params)
// 		if err != nil {
// 			panic(err)
// 		}
// 		// 初始化一下，设置一些默认值
// 		// params.Init()
// 		// 保存params的目的是为了计算CacheKey
// 		// 如果没有缓存需求的, params不应该有值
// 		c.Parameters = &params

// 		c.Executor.actions = make([]Action, 0, len(handlers)+1)
// 		c.AddActions(handlers...)
// 		c.AddActions(func(c *Context) fmt.Stringer {
// 			pairs := tabler.Query(c.DB, params)
// 			response := BuildFromRequest(c.Request, true)
// 			response.Add(pairs...)
// 			c.Responder = response

// 			return bytes.NewBuffer(response.JSON())
// 		})
// 	}
// }
