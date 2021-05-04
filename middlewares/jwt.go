package middlewares

import (
	"strings"
	"sync"

	"github.com/eruca/bisel/btypes"
)

const PairKeyJWT = "JWT"

// 使用的话，必须使用一个jwtSession来构造中间件
func JwtAuthorize(jwtSession btypes.Defaulter) (func(btypes.WhiteList) btypes.Action, btypes.Action) {
	var jwtSessionPool = sync.Pool{
		New: func() interface{} {
			return jwtSession.Default()
		},
	}

	return func(check btypes.WhiteList) btypes.Action {
			return func(c *btypes.Context) (result btypes.PairStringer) {
				if check(c.Parameters) {
					c.Next()

					// 如果在白名单内就直接跳过
					result.Key = PairKeyJWT + " 白名单"
					result.Value = btypes.ValueString(c.Request.Type)
					return
				}

				return validate(c, &jwtSessionPool)
			}
		}, func(c *btypes.Context) btypes.PairStringer {
			return validate(c, &jwtSessionPool)
		}
}

func validate(c *btypes.Context, jwtSessionPool *sync.Pool) btypes.PairStringer {
	var token string
	if c.ConnectionType == btypes.HTTP {
		if v := c.HttpReq.Header.Get("Authorization"); len(v) > 7 && strings.ToLower(v[:6]) == "bearer" {
			token = v[7:]
			return parse(c, token, jwtSessionPool)
		}
	}

	if c.Request.Token == "" {
		c.Responder = btypes.BuildErrorResposeFromRequest(c.ConfigResponseType,
			c.Request, btypes.ErrInvalidToken)
		return btypes.PairStringer{Key: PairKeyJWT, Value: btypes.ValueString(btypes.ErrInvalidToken.Error())}
	}
	return parse(c, c.Request.Token, jwtSessionPool)
}

func parse(c *btypes.Context, token string, jwtSessionPool *sync.Pool) btypes.PairStringer {
	sess := jwtSessionPool.Get().(btypes.Defaulter)
	err := btypes.ParseToken(token, sess)
	if err != nil {
		c.Responder = btypes.BuildErrorResposeFromRequest(c.ConfigResponseType, c.Request, err)
		return btypes.PairStringer{Key: PairKeyJWT, Value: btypes.ValueString(err.Error())}
	}
	c.JwtSession = sess
	c.Next()

	jwtSessionPool.Put(sess)
	return btypes.PairStringer{Key: PairKeyJWT, Value: btypes.ValueString("JWT authority success")}
}
