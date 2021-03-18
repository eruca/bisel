package middlewares

import (
	"fmt"
	"time"

	"github.com/eruca/bisel/types"
)

func TimeElapsed(c *types.Context) fmt.Stringer {
	now := time.Now()
	c.Next()
	duration := time.Since(now)

	return duration
}
