package manager

import (
	"fmt"

	"github.com/eruca/bisel/btypes"
	"github.com/eruca/bisel/net/ws"
)

func (manager *Manager) workflowSuccess(ctx *btypes.Context, paramType btypes.ParamType,
	client *ws.Client, req *btypes.Request) (err error) {

	userid := ctx.JwtSession.UserID()

	switch paramType {
	// 登录请求， 必须是访问了数据库，并且通过验证, 才能与Cacher中的数据进行对比
	case btypes.ParamLogin:
		client.Userid = userid

		data, ok := ctx.Cacher.Get(userid)
		if !ok {
			ctx.Cacher.Set(userid, &UserRuntimeData{
				UserID: userid,
				Client: client,
			})
			return
		}

		userData, ok := data.(*UserRuntimeData)
		if !ok {
			ctx.Logger.Errorf("存储的信息不是 *UserRuntimeData")
			panic("存储的信息不是 *UserRuntimeData")
		}

		// 如果未退出的情况下，有可能出现该连接已经断开
		userData.Client.Send <- btypes.NewRawResponseText(manager.crt, "users/logout", "", []byte("{}")).JSON()
		// 使用新对象替代老对象，避免数据竞争
		ctx.Cacher.Set(userid, &UserRuntimeData{UserID: userid, Client: client})

	case btypes.ParamLogout:
		// Logout 没有内部的操作
		// 实际上登录患者运行时的数据存储在Cacher里user_id => UserRuntimeData
		// 进行清除工作
		if !ctx.Cacher.Remove(userid) {
			ctx.Logger.Errorf("%d 不在Cache内", userid)
			panic("logout 失败")
		}

	case btypes.ParamEditOn:
		key := fmt.Sprintf("%s/%d", ctx.TableName(), ctx.Tabler.Model().ID)
		if _, ok := ctx.Cacher.Get(key); ok {
			err = btypes.ErrTableIsOnEditting
			break
		} else {
			ctx.Cacher.Set(key, struct{}{})
		}

		loginer_id := ctx.JwtSession.UserID()
		v, ok := ctx.Cacher.Get(loginer_id)
		if !ok {
			ctx.Logger.Errorf("%s:%d 不在Cache内", ctx.TableName(), loginer_id)
			panic("用户不在Cache内")
		}
		urd, ok := v.(*UserRuntimeData)
		if !ok {
			ctx.Logger.Errorf("%s:%d存储的不是*UserRuntimeData", ctx.TableName(), loginer_id)
			panic("存储的数据不是*UserRuntimeData")
		}
		if urd.TableName != "" || urd.TableID > 0 {
			err = btypes.ErrTableIsOnEditting
			break
		}
		urd.TableName = ctx.TableName()
		urd.TableID = ctx.Tabler.Model().ID

	case btypes.ParamEditOff:
		key := fmt.Sprintf("%s/%d", ctx.TableName(), ctx.Tabler.Model().ID)
		if _, ok := ctx.Cacher.Get(key); !ok {
			err = btypes.ErrTableIsOffEditting
			break
		} else {
			ctx.Cacher.Remove(key)
		}

		loginer_id := ctx.JwtSession.UserID()
		v, ok := ctx.Cacher.Get(loginer_id)
		if !ok {
			ctx.Logger.Errorf("%s:%d 不在Cache内", ctx.TableName(), loginer_id)
			panic("用户不在Cache内")
		}
		urd, ok := v.(*UserRuntimeData)
		if !ok {
			ctx.Logger.Errorf("%s:%d存储的不是*UserRuntimeData", ctx.TableName(), loginer_id)
			panic("存储的数据不是*UserRuntimeData")
		}
		if urd.TableName == "" || urd.TableID == 0 {
			err = btypes.ErrTableIsOffEditting
			break
		}
		urd.TableName = ""
		urd.TableID = 0
	}

	return
}
