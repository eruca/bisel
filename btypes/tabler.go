package btypes

import (
	"encoding/json"
	"log"
)

type Tabler interface {
	New() Tabler
	FromRequest(json.RawMessage) Tabler

	TableName() string
	Model() *GormModel
	MustAutoMigrate(*DB)
	Register(map[string]ContextConfig)
	// 处理错误：
	// err, true: 如果是处理过的错误要返回给客户端
	// err, false: 意外的错误
	Dispose(error) (bool, error)

	Upsert(*DB, *ParamsContext, Defaulter) (Pairs, error)
	// Query 对于该表进行查询
	// @params: 代表查询的参数
	// return string: 代表该返回在Payload里的key
	// return interface{}: 代表该返回key对应的结果
	Query(*DB, *ParamsContext, Defaulter) (Pairs, error)
	Delete(*DB, *ParamsContext, Defaulter) (Pairs, error)
}

func FromRequestPayload(rw json.RawMessage, tabler Tabler) Tabler {
	err := json.Unmarshal(rw, tabler)
	if err != nil {
		panic(err)
	}
	return tabler
}

type Connectter interface {
	Connected(*DB, Cacher, ConfigResponseType) Responder
}

func DoConnected(db *DB, cacher Cacher, tabler Tabler, cft ConfigResponseType, action string, fullSize bool) Responder {
	pc := ParamsContextForConnectter(fullSize)

	// key是按照查询参数MD5计算出俩的hash值
	request_type := tabler.TableName() + "/" + action
	key := pc.QueryParams.BuildCacheKey(request_type)
	data, ok := cacher.Get(key)
	if ok {
		rb := NewRawResponse(cft, &Request{Type: request_type}, data)
		log.Println("Use Cache:", string(rb.JSON()))
		return rb
	}

	pairs, err := tabler.Query(db, &pc, nil)
	if err != nil {
		panic(err)
	}

	resp := Response{
		Type: cft(request_type, true),
	}
	resp.Add(pairs...)

	// 进入缓存系统
	// 设置缓存
	cacher.Set(tabler.TableName(), key, resp.CachePayload())

	log.Println("Query Database:", string(resp.JSON()))

	return &resp
}
