package middlewares

import (
	"bytes"
	"strings"

	"github.com/eruca/bisel/btypes"
)

const PairKeyJWT = "JWT"

// 使用的话，必须使用一个jwtSession来构造中间件
func JwtAuthorize(jwtSession interface{}) btypes.Action {
	return func(c *btypes.Context) (result btypes.PairStringer) {
		var token string

		if c.ConnectionType == btypes.HTTP {
			if v := c.HttpReq.Header.Get("Authorization"); len(v) > 7 && strings.ToLower(v[:6]) == "bearer" {
				token = v[7:]
				return parse(c, token, jwtSession)
			}
		}

		if c.Request.Token == "" {
			c.Responder = btypes.BuildErrorResposeFromRequest(c.ConfigResponseType,
				c.Request, btypes.ErrInvalidToken)
			return btypes.PairStringer{Key: PairKeyJWT, Value: btypes.ValueString(btypes.ErrInvalidToken.Error())}
		}
		return parse(c, c.Request.Token, jwtSession)
	}
}

func parse(c *btypes.Context, token string, jwtSession interface{}) btypes.PairStringer {
	err := btypes.ParseToken(token, jwtSession)
	if err != nil {
		c.Responder = btypes.BuildErrorResposeFromRequest(c.ConfigResponseType, c.Request, err)
		return btypes.PairStringer{Key: PairKeyJWT, Value: bytes.NewBufferString(err.Error())}
	}
	c.JwtSession = jwtSession
	c.Next()
	return btypes.PairStringer{Key: PairKeyJWT, Value: btypes.ValueString("JWT authority success")}
}
