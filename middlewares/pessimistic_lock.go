package middlewares

import (
	"encoding/json"
	"fmt"

	"github.com/eruca/bisel/btypes"
)

const pessimisticLockKey = "Pessimistic Lock"

var _ btypes.Parameter = (*PessimisticLockParameter)(nil)

type PessimisticLockParameter struct {
	TableName         string `json:"table_name,omitempty"`
	ID                int    `json:"id,omitempty"`
	UserID            int    `json:"user_id,omitempty"`
	Lock              bool   `json:"lock,omitempty"`
	pessimisticTables map[string]struct{}
}

func (*PessimisticLockParameter) String() string { return pessimisticLockKey }
func (pl *PessimisticLockParameter) FromRawMessage(_ btypes.Tabler, rm json.RawMessage) error {
	err := json.Unmarshal(rm, pl)
	if err != nil {
		return err
	}
	return nil
}
func (*PessimisticLockParameter) Status() btypes.RequestStatus { return btypes.StatusNoop }
func (*PessimisticLockParameter) ReadForceUpdate() bool        { return false }
func (*PessimisticLockParameter) BuildCacheKey(string) string  { return "" }
func (*PessimisticLockParameter) JwtCheck() bool               { return true }

// c.Tabler == VirtualTable
func (pl *PessimisticLockParameter) Call(c *btypes.Context, _ btypes.Tabler) (result btypes.Result, err error) {
	// 如果Tabler已经设置为乐观锁，直接返回成功
	if _, ok := pl.pessimisticTables[pl.TableName]; !ok {
		return
	}
	c.Logger.Warnf("RemoteAddr: %q", c.HttpReq.RemoteAddr)

	key := fmt.Sprintf("%s/%d", pl.TableName, pl.ID)
	userid, ok := c.Cacher.Get(key)
	if !ok {
		if pl.Lock {
			result.Payloads.Push("msg", fmt.Sprintf("已获取%q写锁", key))
			c.Logger.Infof("%d: 获取%s", pl.UserID, key)

			c.Cacher.Set(key, pl.UserID)
			c.Cacher.Set(pl.UserID, key)
		} else {
			err = fmt.Errorf("%s 未被占用，现在却是要求解锁，你那里写错了", key)
			c.Logger.Errorf(err.Error())
		}
	} else {
		if pl.Lock {
			err = fmt.Errorf("%s 已被 %d 占用，现在却是要求上锁，你哪里写错了", key, userid)
			c.Logger.Errorf(err.Error())
		} else {
			c.Cacher.Remove(key)
			c.Cacher.Remove(userid)
			result.Payloads.Push("msg", fmt.Sprintf("删除%q写锁", key))
			c.Logger.Infof("%d: 删除%q", pl.UserID, key)
		}
	}
	return
}

func PessimisticLockHandler(pess map[string]struct{}, actions ...btypes.Action) btypes.ContextConfig {
	return btypes.HandlerFunc(&btypes.VirtualTable{}, &PessimisticLockParameter{pessimisticTables: pess}, nil, actions...)
}
