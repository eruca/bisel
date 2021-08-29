package btypes

import (
	"github.com/eruca/bisel/logger"
)

// ***********************************************************
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
	Register(map[string]ContextConfig)
	// Depends 代表该表需要依赖其他表
	// 所以如果依赖表发生该表，则该表就需要删除缓存
	Depends() []string

	// 对于Gorm的配置
	TableName() string
	// 默认是乐观锁，可以设置为悲观锁
	PessimisticLock() bool
	// 对于每个Table的默认设置
	Size() int
	Orderby() string

	Model() *GormModel

	Query(*Context, Tabler, *QueryParam, JwtSession) (Result, error)
	// 查询时剔除的列
	QueryOmits() []string

	Insert(*Context, Tabler, JwtSession) (Result, error)
	Delete(*Context, Tabler, JwtSession) (Result, error)
	Update(*Context, Tabler, JwtSession) (Result, error)
}

type Connectter interface {
	Push(*DB, Cacher, logger.Logger, ConfigResponseType) Responder
}

func DefaultPush(db *DB, cacher Cacher, log logger.Logger, crt ConfigResponseType,
	tabler Tabler, action string) Responder {

	tableName := tabler.TableName()
	qp := QueryParam{
		Size:    int64(tabler.Size()),
		Orderby: tabler.Orderby(),
	}
	ctx := Context{DB: db, Cacher: cacher, Logger: log, ConfigResponseType: crt}
	request_type := tableName + "/" + action
	key := qp.BuildCacheKey(request_type)

	bin := cacher.GetBucket(tableName, key)
	if bin != nil {
		rb := NewRawResponseText(crt, request_type, "", bin)
		log.Infof("Use Cache: %s", string(rb.JSON()))
		return rb
	}

	result, err := tabler.Query(&ctx, tabler, &qp, nil)
	if err != nil {
		panic(err)
	}

	resp := &Response{
		Type:      crt(request_type, true),
		broadcast: result.Broadcast,
	}
	resp.Add(result.Payloads...)

	// 设置缓存
	cacher.SetBucket(tableName, key, resp.JSONPayload())
	log.Infof("Push => Query Database & Set Cache: %s", string(resp.JSON()))
	return resp
}
