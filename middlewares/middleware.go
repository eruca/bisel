package middlewares

import "github.com/eruca/bisel/btypes"

func QueryDefault(useJwt bool) []btypes.Action {
	if useJwt {
		return []btypes.Action{TimeElapsed, JwtAuthorize, UseCache}
	}
	return []btypes.Action{TimeElapsed, UseCache}
}
