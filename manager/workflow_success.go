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
			ctx.Logger.Debugf("查找 userid:%d 失败", userid)
			ctx.Cacher.Set(userid, &UserRuntimeData{
				UserID: userid,
				Client: client,
			})
			return
		}

		ctx.Logger.Debugf("查找 userid:%d 成功", userid)
		userData, ok := data.(*UserRuntimeData)
		if !ok {
			ctx.Logger.Errorf("存储的信息不是 *UserRuntimeData")
			panic("存储的信息不是 *UserRuntimeData")
		}

		// 如果未退出的情况下，有可能出现该连接已经断开
		ctx.Logger.Debugf("发送退出users/logout信号给原客户端")
		userData.Client.Send <- btypes.NewRawResponseText(manager.crt, "users/logout", "", []byte("{}")).JSON()
		// 使用新对象替代老对象，避免数据竞争
		ctx.Logger.Debugf("重新设置userid:%d的登录状态", userid)
		ctx.Cacher.Set(userid, &UserRuntimeData{UserID: userid, Client: client})

	case btypes.ParamLogout:
		// Logout 没有内部的操作
		// 实际上登录患者运行时的数据存储在Cacher里user_id => UserRuntimeData
		// 进行清除工作
		if !ctx.Cacher.Remove(userid) {
			ctx.Logger.Errorf("%d 不在Cache内", userid)
			panic("logout 失败")
		}
		ctx.Logger.Debugf("userid:%d 退出成功", userid)
	case btypes.ParamEditOn:
		key := fmt.Sprintf("%s/%d", ctx.TableName(), ctx.Tabler.Model().ID)
		if _, ok := ctx.Cacher.Get(key); ok {
			err = btypes.ErrTableIsOnEditting
			ctx.Logger.Errorf("%s 已经在编辑中, 发送错误给客户端:%v", key, err)
			break
		} else {
			ctx.Logger.Debugf("在Cacher中Set:%s", key)
			ctx.Cacher.Set(key, struct{}{})
		}

		loginer_id := ctx.JwtSession.UserID()
		v, ok := ctx.Cacher.Get(loginer_id)
		if !ok {
			ctx.Logger.Errorf("userid:%d 不在Cacher内", loginer_id)
			panic("用户不在Cache内")
		}
		ctx.Logger.Debugf("userid:%d 已经在Cacher中了")

		urd, ok := v.(*UserRuntimeData)
		if !ok {
			ctx.Logger.Errorf("%s:%d存储的不是*UserRuntimeData", ctx.TableName(), loginer_id)
			panic("存储的数据不是*UserRuntimeData")
		}
		if urd.TableName != "" || urd.TableID > 0 {
			err = btypes.ErrTableIsOnEditting
			ctx.Logger.Errorf("目前userid:%d 存在编辑信息: %s/%d, 错误信息:%v", userid, urd.TableName, urd.TableID, err)
			break
		}
		urd.TableName = ctx.TableName()
		urd.TableID = ctx.Tabler.Model().ID
		ctx.Logger.Debugf("目前userid:%d 存入编辑信息: %s/%d", userid, urd.TableName, urd.TableID)

	case btypes.ParamEditOff:
		key := fmt.Sprintf("%s/%d", ctx.TableName(), ctx.Tabler.Model().ID)
		if _, ok := ctx.Cacher.Get(key); !ok {
			err = btypes.ErrTableIsOffEditting
			ctx.Logger.Errorf("%s 不在Cacher中, 发送错误:%v", key, err)
			break
		} else {
			ctx.Logger.Debugf("在Cacher中Remove: %s", key)
			ctx.Cacher.Remove(key)
		}

		loginer_id := ctx.JwtSession.UserID()
		v, ok := ctx.Cacher.Get(loginer_id)
		if !ok {
			ctx.Logger.Errorf("%s:%d 不在Cache内", ctx.TableName(), loginer_id)
			panic("用户不在Cache内")
		}
		ctx.Logger.Debugf("userid:%d 已经在Cacher中了")

		urd, ok := v.(*UserRuntimeData)
		if !ok {
			ctx.Logger.Errorf("%s:%d存储的不是*UserRuntimeData", ctx.TableName(), loginer_id)
			panic("存储的数据不是*UserRuntimeData")
		}
		if urd.TableName == "" || urd.TableID == 0 {
			err = btypes.ErrTableIsOffEditting
			ctx.Logger.Errorf("目前userid:%d 不存在编辑信息，错误信息:%v", userid, err)
			break
		}
		ctx.Logger.Debugf("目前userid:%d 清除原来编辑信息: %s/%d", userid, urd.TableName, urd.TableID)
		urd.TableName = ""
		urd.TableID = 0
	}

	return
}
