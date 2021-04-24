package middlewares

import (
	"fmt"

	"github.com/eruca/bisel/btypes"
)

const (
	PairKeyCache = "Use Cache"
)

func UseCache(c *btypes.Context) btypes.PairStringer {
	// 如果没有cache，直接跳过
	if c.Cacher == nil {
		panic("使用了Cache，而cacher却是nil，需设置")
	}

	paramType := c.Parameters.ParamType
	// 如果是write: upsert/delete,需删除缓存数据
	if paramType == btypes.ParamUpsert || paramType == btypes.ParamDelete {
		c.Cacher.Clear(c.TableName())
		c.Next()
		return btypes.PairStringer{Key: PairKeyCache, Value: btypes.ValueString("clear cache:" + c.TableName())}
	}

	params := c.Parameters.QueryParams
	// 如果客户端请求没有Hash这个值或者要求强制走数据库，就是没有缓存过
	// 直接跳过
	if params.ReforceUpdated {
		return noExistInCache(c, params)
	}

	cacheKey := params.BuildCacheKey(c.Request.Type)
	value, ok := c.Cacher.Get(cacheKey)
	if ok {
		c.Responder = btypes.NewRawBytes(value)
		return btypes.PairStringer{Key: PairKeyCache, Value: btypes.ValueString(value)}
	}
	return noExistInCache(c, params)
}

func noExistInCache(c *btypes.Context, params *btypes.QueryParams) btypes.PairStringer {
	// 先进行后面的操作，返回的时候应该已经有Response了
	// 就可以对其进行缓存
	c.Next()
	if c.Responder == nil {
		panic("这个时候应该有对客户端的回应了，可是没有")
	}
	// key是按照查询参数MD5计算出俩的hash值
	key := params.BuildCacheKey(c.Request.Type)
	// 设置缓存
	c.Cacher.Set(c.Tabler.TableName(), key, c.Responder.JSON())

	return btypes.PairStringer{
		Key:   PairKeyCache,
		Value: btypes.ValueString(fmt.Sprintf("rebuild from %s: %v", c.Request.Type, *params)),
	}
}
