package middlewares

import (
	"fmt"
	"strings"

	"github.com/eruca/bisel/btypes"
)

const (
	PairKeyCache = "Flow @Cache"
)

func UseCache(c *btypes.Context) btypes.PairStringer {
	// 如果没有cache，直接跳过
	if c.Cacher == nil {
		panic("使用了Cache，而cacher却是nil，需设置")
	}

	c.Logger.Warnf("Use Cache: %v", c.Parameter.Status())
	switch c.Parameter.Status() {
	case btypes.StatusNoop:
		c.Next()
		return btypes.PairStringer{Key: PairKeyCache, Value: btypes.ValueString("No op")}

	case btypes.StatusRead:
		// params := c.ParamContext.QueryParam
		// 如果客户端请求没有Hash这个值或者要求强制走数据库，就是没有缓存过
		// 直接跳过
		if c.Parameter.ReadForceUpdate() {
			return noExistInCache(c)
		}

		cacheKey := c.BuildCacheKey(c.Request.Type)
		bin := c.Cacher.GetBucket(c.TableName(), cacheKey)
		if bin != nil {
			c.Responder = btypes.NewRawResponse(c.ConfigResponseType, c.Request, bin)
			return btypes.PairStringer{Key: PairKeyCache, Value: btypes.ValueString(bin)}
		}
		return noExistInCache(c)

	case btypes.StatusWrite:
		var builder strings.Builder
		builder.WriteString("clear cache: ")
		tableName := c.TableName()
		builder.WriteString(tableName)

		c.Cacher.ClearBuckets(tableName)
		for key := range c.Depends[tableName] {
			c.Cacher.ClearBuckets(key)
			builder.WriteByte(',')
			builder.WriteString(key)
		}

		c.Next()
		return btypes.PairStringer{Key: PairKeyCache, Value: btypes.ValueString(builder.String())}

	default:
		panic("unknown status")
	}
}

func noExistInCache(c *btypes.Context) btypes.PairStringer {
	// 先进行后面的操作，返回的时候应该已经有Response了
	// 就可以对其进行缓存
	c.Next()
	if c.Responder == nil {
		panic("这个时候应该有对客户端的回应了，可是没有")
	}
	// key是按照查询参数MD5计算出俩的hash值
	key := c.BuildCacheKey(c.Request.Type)
	// 设置缓存
	// 只能缓存payload,如果缓存responder，则会加入uuid
	c.Cacher.SetBucket(c.Tabler.TableName(), key, c.Responder.JSONPayload())

	return btypes.PairStringer{
		Key:   PairKeyCache,
		Value: btypes.ValueString(fmt.Sprintf("rebuild cache from %s: %v", c.Request.Type, c.Parameter)),
	}
}
