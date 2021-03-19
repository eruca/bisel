package btypes

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Model ...
type GormModel struct {
	ID        uint           `gorm:"primarykey" json:"key,omitempty"`
	CreatedAt time.Time      `json:"created_at,omitempty"`
	UpdatedAt time.Time      `json:"-,omitempty"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Version   uint           `json:"version,omitempty" gorm:"default:1"`
}

// RowID 可以直接在DBmodel内部实现
func (model *GormModel) RowID() uint {
	return model.ID
}

// insert 插入新数据时有可能会违反独一约束，则需要处理该类错误，需在tabler内部处理
func (model *GormModel) insert(db *DB, tabler Tabler) error {
	err := db.Gorm.Create(tabler).Error
	if err != nil {
		e := err.Error()
		for _, group := range ErrGroup {
			if strings.Contains(e, group.Key) {
				s := strings.TrimSpace(strings.TrimPrefix(e, group.Key))
				return fmt.Errorf(group.Value, s)
			}
		}
		panic(err)
	}
	return nil
}

// update 数据，直接Save，保存所有数据，同时因为如果version不一致就返回0行，所以是乐观锁错误
func (model *GormModel) update(db *DB, tabler Tabler, omits ...string) error {
	model.Version++
	tx := db.Gorm.Model(tabler).Where("version = ?", model.Version-1).
		Omit("created_at", "deleted_at").Omit(omits...).Updates(tabler)

	if err := tx.Error; err != nil {
		panic(err)
	}
	if tx.RowsAffected == 0 {
		return ErrOptimisticLock
	}
	return nil
}

// Upsert 插入或更新数据库
// Update的情况下，可能有一部分数据不想更新，就可以使用update_omits_columns来跳过，插入则会忽略
// return @bool 表示是不是插入
// return @error 代表返回客户端的错误
func (model *GormModel) Upsert(db *DB, tabler Tabler, update_omits_columns ...string) (bool, error) {
	if model.ID == 0 {
		return true, model.insert(db, tabler)
	}
	return false, model.update(db, tabler, update_omits_columns...)
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
