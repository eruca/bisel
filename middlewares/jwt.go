package middlewares

import (
	"bytes"
	"strings"

	"github.com/eruca/bisel/btypes"
)

const PairKeyJWT = "JWT"

func Parse(c *btypes.Context, token string) btypes.PairStringer {
	claim, err := btypes.ParseToken(token)
	if err != nil {
		c.Responder = btypes.BuildErrorResposeFromRequest(c.ConfigResponseType, c.Request, err)
		return btypes.PairStringer{Key: PairKeyJWT, Value: bytes.NewBufferString(err.Error())}
	}
	c.ClaimContent = &claim.ClaimContent
	c.Next()
	return btypes.PairStringer{Key: PairKeyJWT, Value: bytes.NewBufferString("JWT authority success")}
}

func JwtAuthorize(c *btypes.Context) (result btypes.PairStringer) {
	var token string

	if c.ConnectionType == btypes.HTTP {
		if v := c.HttpReq.Header.Get("Authorization"); len(v) > 7 && strings.ToLower(v[:6]) == "bearer" {
			token = v[7:]
			return Parse(c, token)
		}
	}

	if c.Request.Token == "" {
		c.Responder = btypes.BuildErrorResposeFromRequest(c.ConfigResponseType,
			c.Request, btypes.ErrInvalidToken)
		return btypes.PairStringer{Key: PairKeyJWT, Value: bytes.NewBufferString(btypes.ErrInvalidToken.Error())}
	}
	return Parse(c, c.Request.Token)
}
