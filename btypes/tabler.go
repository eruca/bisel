package btypes

import (
	"encoding/json"
)

type Tabler interface {
	Model() *GormModel
	FromRequest(json.RawMessage) Tabler
	TableName() string
	Upsert(*DB, *ParamsContext) (Pairs, error)
	Delete(*DB, *ParamsContext) (Pairs, error)
	New() Tabler

	// Query 对于该表进行查询
	// @params: 代表查询的参数
	// return string: 代表该返回在Payload里的key
	// return interface{}: 代表该返回key对应的结果
	Query(*DB, *ParamsContext) (Pairs, error)
	MustAutoMigrate(*DB)

	// Connected(*Context)
	Register(map[string]ContextConfig)
	// 处理错误：
	// err, true: 如果是处理过的错误要返回给客户端
	// err, false: 意外的错误
	Dispose(error) (bool, error)
}

func FromRequestPayload(rw json.RawMessage, tabler Tabler) Tabler {
	err := json.Unmarshal(rw, tabler)
	if err != nil {
		panic(err)
	}
	return tabler
}

type Connectter interface {
	Connected(*DB, Cacher) Responder
}
