package manager

import (
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/eruca/bisel/types"
)

type Manager struct {
	db                 *types.DB
	cacher             types.Cacher
	tablers            []types.Tabler
	handlers           map[string]types.ContextConfig
	configResponseType types.ConfigResponseType
}

// New ...
func New(db *types.DB, cacher types.Cacher,
	configResponseType types.ConfigResponseType, tablers ...types.Tabler) *Manager {

	handlers := make(map[string]types.ContextConfig)
	for _, tabler := range tablers {
		tabler.MustAutoMigrate(db)
		// Register实际上是让tabler自己注册数据到handlers上
		tabler.Register(handlers)
	}
	// 如果未设置，使用默认的
	if configResponseType == nil {
		configResponseType = types.DefaultResponseType
	}

	manager := &Manager{
		db:                 db,
		cacher:             cacher,
		tablers:            tablers,
		handlers:           handlers,
		configResponseType: configResponseType,
	}

	return manager
}

// TakeAction 可以并发执行
//! Notice: 因为写入都是在初始化阶段，读取可以并发
func (manager *Manager) TakeAction(clientWriter io.Writer, req *types.Request, httpReq *http.Request) (err error) {
	var respReader io.Reader

	if handler, ok := manager.handlers[req.Type]; ok {
		ctx := types.NewContext(manager.db, manager.cacher, req, httpReq, manager.configResponseType)
		handler(ctx)
		ctx.Start()

		if ctx.Responder == nil {
			panic("需要返回一个结果给客户端")
		}
		respReader = types.ResponderToReader(ctx.Responder)
		log.Printf("各个handler结果:\t%v\n", ctx.Results)
	} else {
		resp := types.BuildErrorResposeFromRequest(manager.configResponseType, req, fmt.Errorf("%q router not implemented yet", req.Type))
		respReader = types.ResponderToReader(resp)
	}
	_, err = io.Copy(clientWriter, respReader)
	return
}

// Connected 当连接建立时
func (manager *Manager) Connected(c chan<- []byte) {
	for _, tabler := range manager.tablers {
		if connecter, ok := tabler.(types.Connectter); ok {
			responder := connecter.Connected(manager.DB, manager.Cacher)
			c <- responder.JSON()
		}
	}
}
