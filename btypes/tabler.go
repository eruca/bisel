package btypes

import (
	"encoding/json"
)

// Result 代表增删查修的结果
// Payloads键值对是返回给客户端的数据
// Broadcast 代表是否进行广播
type Result struct {
	Payloads  Pairs
	Broadcast bool
}

type Tabler interface {
	New() Tabler
	Done()
	FromRequest(json.RawMessage) Tabler

	TableName() string
	Model() *GormModel
	MustAutoMigrate(*DB)
	Register(map[string]ContextConfig)
	// 处理错误：
	// err, true: 如果是处理过的错误要返回给客户端
	// err, false: 意外的错误
	Dispose(error) (bool, error)

	Upsert(*DB, *ParamsContext, Defaulter) (Result, error)
	// Query 对于该表进行查询
	// @ParamsContext: 代表查询的参数
	// @Defaulter代表JWT中的权限
	// @Result 代表返回的Key=>Value对，同时表明是否是broadcast
	// @error 代表发生错误，返回给客户端的信息
	Query(*DB, *ParamsContext, Defaulter) (Result, error)
	Delete(*DB, *ParamsContext, Defaulter) (Result, error)
}

func FromRequestPayload(rw json.RawMessage, tabler Tabler) Tabler {
	err := json.Unmarshal(rw, tabler)
	if err != nil {
		panic(err)
	}
	return tabler
}

// 推送时机
type PushTimer uint8

const (
	Connected PushTimer = iota
	Logined
)

type Pusher interface {
	Connectter
	When() PushTimer
	Auth(Defaulter) bool
}

type Connectter interface {
	Push(*DB, Cacher, ConfigResponseType, Logger) Responder
}

func PushWithDefaultParameter(db *DB, cacher Cacher, cft ConfigResponseType, logger Logger,
	tabler Tabler, action string, fullSize bool) Responder {
	pc := ParamsContextForConnectter(fullSize)

	// key是按照查询参数MD5计算出俩的hash值
	request_type := tabler.TableName() + "/" + action
	key := pc.QueryParams.BuildCacheKey(request_type)
	data, ok := cacher.Get(key)
	if ok {
		rb := NewRawResponseText(cft, request_type, "", data)
		logger.Infof("Use Cache: %s", string(rb.JSON()))
		return rb
	}

	result, err := tabler.Query(db, &pc, nil)
	if err != nil {
		panic(err)
	}

	resp := newResponse()
	resp.Type = cft(request_type, true)
	resp.broadcast = result.Broadcast

	resp.Add(result.Payloads...)

	// 进入缓存系统
	// 设置缓存
	cacher.Set(tabler.TableName(), key, resp.CachePayload())

	logger.Infof("Query Database: %s", string(resp.JSON()))

	return resp
}
