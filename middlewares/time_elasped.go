package middlewares

import (
	"time"

	"github.com/eruca/bisel/btypes"
)

func TimeElapsed(c *btypes.Context) btypes.PairStringer {
	now := time.Now()
	c.Next()
	duration := time.Since(now)

	return btypes.PairStringer{Key: "Time Elapsed", Value: duration}
}
