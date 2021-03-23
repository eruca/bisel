package middlewares

import (
	"bytes"
	"fmt"

	"github.com/eruca/bisel/btypes"
)

func UseCache(c *btypes.Context) fmt.Stringer {
	// 如果没有cache，直接跳过
	if c.Cacher == nil {
		panic("使用了Cache，而cacher却是nil，需设置")
	}

	paramType := c.Parameters.ParamType
	// 如果是write: upsert/delete,需删除缓存数据
	if paramType == btypes.ParamUpsert || paramType == btypes.ParamDelete {
		c.Cacher.Clear(c.TableName())
		c.Next()
		return nil
	}

	params := c.Parameters.QueryParams
	// 如果客户端请求没有Hash这个值或者要求强制走数据库，就是没有缓存过
	// 直接跳过
	if params.ReforceUpdated {
		noExistCache(c, params)
		return nil
	}

	// 发送过来的请求没有Hash
	if params.Hash == "" {
		cacheKey := params.BuildCacheKey(c.Request)
		value, ok := c.Cacher.Get(cacheKey)
		if ok {
			c.Responder = btypes.NewRawBytes(value)
			return bytes.NewBuffer(value)
		} else {
			noExistCache(c, params)
			return nil
		}
	}

	// 发送过来的数据有Hash
	_, ok := c.Cacher.Get(params.Hash)
	// 如果客户端请求有Hash，可是在缓存中无法找到
	// 表示缓存已经被删除
	if !ok {
		noExistCache(c, params)
		return nil
	}

	// 如果缓存有数据，就直接返回给客户端了
	// 那么后面的actions都不执行了，如果有想要执行的action必须在缓存之前
	//! 查询了缓存后，不需要把Hash再一次给客户端，原来的Hash值还是有效的, 也不需要给数据，因为客户端有了
	c.Responder = btypes.BuildFromRequest(c.ConfigResponseType, c.Request, true)
	return bytes.NewBufferString("response without payload")
}

func noExistCache(c *btypes.Context, params *btypes.QueryParams) {
	// 先进行后面的操作，返回的时候应该已经有Response了
	// 就可以对其进行缓存
	c.Next()
	if c.Responder == nil {
		panic("这个时候应该有对客户端的回应了，可是没有")
	}
	// key是按照查询参数MD5计算出俩的hash值
	key := params.BuildCacheKey(c.Request)
	// 设置缓存
	c.Cacher.Set(c.Tabler.TableName(), key, c.Responder.JSON())
	// 给返回的结果增加Hash值，下次请求带上这个哈希值，就可以使用缓存了
	c.Responder.AddHash(key)
}
