package manager

import (
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/eruca/bisel/types"
)

type Manager struct {
	*types.DB
	types.Cacher
	tablers  []types.Tabler
	handlers map[string]types.ContextConfig
}

// New ...
func New(db *types.DB, cacher types.Cacher, tablers ...types.Tabler) *Manager {
	handlers := make(map[string]types.ContextConfig)
	for _, tabler := range tablers {
		tabler.MustAutoMigrate(db)
		// Register实际上是让tabler自己注册数据到handlers上
		tabler.Register(handlers)
	}

	manager := &Manager{
		DB:       db,
		Cacher:   cacher,
		tablers:  tablers,
		handlers: handlers,
	}

	return manager
}

// TakeAction 可以并发执行
//! Notice: 因为写入都是在初始化阶段，读取可以并发
func (manager *Manager) TakeAction(clientWriter io.Writer, req *types.Request, httpReq *http.Request) (err error) {
	var respReader io.Reader

	if handler, ok := manager.handlers[req.Type]; ok {
		ctx := types.NewContext(manager.DB, manager.Cacher, req, httpReq)
		handler(ctx)
		ctx.Start()

		if ctx.Responder == nil {
			panic("需要返回一个结果给客户端")
		}
		respReader = types.ResponderToReader(ctx.Responder)
		log.Printf("各个handler结果:\t%v\n", ctx.Results)
	} else {
		resp := types.BuildErrorResposeFromRequest(req, fmt.Errorf("%q router not implemented yet", req.Type))
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
