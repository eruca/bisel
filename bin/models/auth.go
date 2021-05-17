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
	ID   uint `json:"id,omitempty"`
	Role uint `json:"role,omitempty"`
}

func (js *JwtSession) Default() btypes.Defaulter {
	return &JwtSession{}
}

func (js *JwtSession) UserID() uint {
	return js.ID
}
