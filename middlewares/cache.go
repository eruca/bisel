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

	// 发送过来的请求没有Hash
	if params.Hash == "" {
		cacheKey := params.BuildCacheKey(c.Request.Type)
		value, ok := c.Cacher.Get(cacheKey)
		if ok {
			rb := btypes.NewRawBytes(value)
			rb.AddHash(cacheKey)
			c.Responder = rb
			return btypes.PairStringer{Key: PairKeyCache, Value: btypes.ValueString(value)}
		} else {
			return noExistInCache(c, params)
		}
	}

	// 发送过来的数据有Hash
	_, ok := c.Cacher.Get(params.Hash)
	// 如果客户端请求有Hash，可是在缓存中无法找到
	// 表示缓存已经被删除
	if !ok {
		return noExistInCache(c, params)
	}

	// 如果缓存有数据，就直接返回给客户端了
	// 那么后面的actions都不执行了，如果有想要执行的action必须在缓存之前
	//! 查询了缓存后，不需要把Hash再一次给客户端，原来的Hash值还是有效的, 也不需要给数据，因为客户端有了
	c.Responder = btypes.BuildFromRequest(c.ConfigResponseType, c.Request, true)
	return btypes.PairStringer{Key: PairKeyCache, Value: btypes.ValueString("response without payload")}
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
	// 给返回的结果增加Hash值，下次请求带上这个哈希值，就可以使用缓存了
	c.Responder.AddHash(key)
	// 设置缓存
	c.Cacher.Set(c.Tabler.TableName(), key, c.Responder.JSON())

	return btypes.PairStringer{
		Key:   PairKeyCache,
		Value: btypes.ValueString(fmt.Sprintf("rebuild from %s: %v", c.Request.Type, *params)),
	}
}
