package middlewares

import (
	"fmt"
	"time"

	"github.com/eruca/bisel/btypes"
)

func TimeElapsed(c *btypes.Context) fmt.Stringer {
	now := time.Now()
	c.Next()
	duration := time.Since(now)

	return duration
}
