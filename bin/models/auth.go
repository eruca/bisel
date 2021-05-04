package models

import (
	"github.com/eruca/bisel/btypes"
	"github.com/eruca/bisel/middlewares"
)

var (
	_                     btypes.Defaulter = (*JwtSession)(nil)
	JwtAuthCheck, JwtAuth                  = middlewares.JwtAuthorize(&JwtSession{})
)

type JwtSession struct {
	Role uint `json:"role,omitempty"`
}

func (js *JwtSession) Default() btypes.Defaulter {
	return &JwtSession{}
}
