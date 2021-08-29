package btypes

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Model ...
type GormModel struct {
	ID        uint           `json:"id,omitempty" gorm:"primarykey"`
	CreatedAt time.Time      `json:"created_at,omitempty"`
	UpdatedAt time.Time      `json:"updated_at,omitempty"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
	Version   uint           `json:"version,omitempty" gorm:"default:1"`
}

// RowID 可以直接在DBmodel内部实现
func (model *GormModel) RowID() uint { return model.ID }
func (*GormModel) Done()             {}

// todo PessimisticLock 乐观锁/悲观锁需推送给客户端
func (*GormModel) PessimisticLock() bool { return false }
func (*GormModel) Depends() []string     { return nil }
func (*GormModel) Size() int             { return 20 }
func (*GormModel) Orderby() string       { return "updated_at DESC" }

// update 数据，直接Save，保存所有数据，同时因为如果version不一致就返回0行，所以是乐观锁错误
func (model *GormModel) UpdateWithOmits(db *DB, tabler Tabler, omits ...string) error {
	model.Version++
	tx := db.Gorm.Model(tabler).Where("version = ?", model.Version-1).
		Omit("deleted_at").Omit(omits...).Updates(tabler)

	if err := tx.Error; err != nil {
		panic(err)
	}
	if tx.RowsAffected == 0 {
		return ErrOptimisticLock
	}
	return nil
}

// Delete 因为Gorm提供了软删除与硬删除
func (model *GormModel) delete(db *DB, tabler Tabler, hardDelete bool) (int64, error) {
	tx := db.Gorm
	if hardDelete {
		tx = db.Gorm.Unscoped()
	}

	tx = tx.Where("version = ?", model.Version).Delete(tabler)
	if err := tx.Error; err != nil {
		panic(err)
	}
	if tx.RowsAffected == 0 {
		return 0, ErrOptimisticLock
	}

	return tx.RowsAffected, nil
}

func (model *GormModel) HardDelete(db *DB, tabler Tabler) (int64, error) {
	return model.delete(db, tabler, true)
}

func (model *GormModel) SoftDelete(db *DB, tabler Tabler) (int64, error) {
	return model.delete(db, tabler, false)
}

func (model *GormModel) Model() *GormModel { return model }

// insert 插入新数据时有可能会违反独一约束，则需要处理该类错误，需在tabler内部处理
func (*GormModel) Insert(c *Context, tabler Tabler, jwtSess JwtSession) (result Result, err error) {
	if err = c.DB.Gorm.Create(tabler).Error; err == nil {
		result.Payloads.Push("msg", "添加成功")
		return
	}

	if strings.Contains(err.Error(), ErrStringUniqueConstrait) {
		return
	}
	panic(err)
}

func (model *GormModel) Delete(c *Context, tabler Tabler, jwtSession JwtSession) (result Result, err error) {
	c.Logger.Infof("delete %#v", tabler)

	if tabler.PessimisticLock() {
		writeLockKey := fmt.Sprintf("%s/%d", tabler.TableName(), model.ID)
		userid, ok := c.Cacher.Get(writeLockKey)
		if ok {
			err = fmt.Errorf("目前该数据%q已被其他客户端:%v锁住，不能删除", writeLockKey, userid)
			return
		}
	}

	var n int64
	n, err = tabler.Model().SoftDelete(c.DB, tabler)
	if err == nil {
		result.Payloads.Push("msg", fmt.Sprintf("成功删除[%d]", n))
	}
	return
}

func (*GormModel) Update(c *Context, tabler Tabler, jwtSess JwtSession) (result Result, err error) {
	err = tabler.Model().UpdateWithOmits(c.DB, tabler)
	if err != nil {
		return
	}
	result.Payloads.Push("msg", "修改成功")
	result.Payloads.Push("tabler", tabler)
	return
}

func (model *GormModel) QueryOmits() []string { return nil }

func (*GormModel) Query(c *Context, tabler Tabler, query *QueryParam,
	jwtSess JwtSession) (result Result, err error) {

	if query == nil {
		panic("参数不能为空")
	}

	c.Logger.Infof("tabler in Query: %v, %s", tabler, tabler.TableName())

	typeTabler := reflect.TypeOf(tabler)
	for typeTabler.Kind() == reflect.Ptr {
		typeTabler = typeTabler.Elem()
	}
	size := 20
	if tabler.Size() > 0 {
		size = tabler.Size()
	}
	tablerSlice := reflect.MakeSlice(reflect.SliceOf(typeTabler), 0, size)

	ptr := reflect.New(tablerSlice.Type())
	ptr.Elem().Set(tablerSlice)

	var total int64
	QueryAssist(c.DB.Gorm, tabler, query, &total, ptr.Interface(), tabler.QueryOmits()...)

	result.Payloads.Push("total", total)
	result.Payloads.Push("list", ptr.Interface())
	return
}
