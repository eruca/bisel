package btypes

import "bytes"

type Action func(c *Context) PairStringer

// 对于Context进行配置
// @return represent islogin
type ContextConfig func(*Context)

// ConfigAction 自己设置最后一个Action
// type ConfigAction func(*Context, Tabler, ParamContext, JwtSession) (Result, string, error)

// jwtSession: 目的是将jwt的需求构造成一个结构体，发送给客户端就可以里，这个Context也完成使命被回收了
func HandlerFunc(tabler Tabler, parameter Parameter, jwtSession JwtSession, handlers ...Action) ContextConfig {
	return func(c *Context) {
		tabler = tabler.New()

		parameter.FromRawMessage(tabler, c.Request.Payload)
		c.fill(tabler, parameter, handlers...)
		if jwtSession != nil {
			c.JwtSess = jwtSession
		}

		c.AddActions(func(ctx *Context) PairStringer {
			result, err := ctx.Parameter.Call(ctx, tabler)
			response := ctx.BuildResponse(result, err)
			ctx.Logger.Infof("HandlerFunc: %s", ctx.Parameter)

			return PairStringer{Key: ctx.Parameter.String(), Value: bytes.NewBuffer(response.JSON())}
		})
	}
}

func QueryHandler(tabler Tabler, checkJwt bool, handlers ...Action) ContextConfig {
	return HandlerFunc(tabler, &QueryParameter{CheckJWT: checkJwt}, nil, handlers...)
}

func InsertHandler(tabler Tabler, checkJwt bool, handlers ...Action) ContextConfig {
	return HandlerFunc(tabler, &WriterParameter{
		ParamType: ParamInsert,
		CheckJWT:  checkJwt,
	}, nil, handlers...)
}

func UpdateHandler(tabler Tabler, checkJwt bool, handlers ...Action) ContextConfig {
	return HandlerFunc(tabler, &WriterParameter{
		ParamType: ParamUpdate,
		CheckJWT:  checkJwt,
	}, nil, handlers...)
}

func DeleteHandler(tabler Tabler, checkJwt bool, handlers ...Action) ContextConfig {
	return HandlerFunc(tabler, &WriterParameter{
		ParamType: ParamDelete,
		CheckJWT:  checkJwt,
	}, nil, handlers...)
}
