package journal

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/eruca/bisel/btypes"
	"github.com/eruca/bisel/middlewares"
	"gorm.io/gorm"
)

var (
	_         btypes.Tabler = (*Journal)(nil)
	tableName string        = "journals"
)

type Journal struct {
	btypes.GormModel
	Name        string            `json:"name,omitempty"`
	Url         string            `json:"url,omitempty"`
	Description btypes.NullString `json:"description,omitempty"`
}

func (j *Journal) New() btypes.Tabler {
	return &Journal{}
}

func (j *Journal) Model() *btypes.GormModel {
	return &j.GormModel
}

func (j *Journal) Register(handlers map[string]btypes.ContextConfig) {
	// 这里的j实际上是Manager.New时传入的对象
	// 这个对象应该一直都是空值，只是作为调用
	handlers[tableName+"/query"] = btypes.HandlerFunc(j, btypes.ParamQuery,
		middlewares.TimeElapsed, middlewares.UseCache)
	handlers[tableName+"/upsert"] = btypes.HandlerFunc(j, btypes.ParamUpsert, middlewares.TimeElapsed)
	handlers[tableName+"/delete"] = btypes.HandlerFunc(j, btypes.ParamDelete, middlewares.TimeElapsed)
}

func (j *Journal) MustAutoMigrate(db *btypes.DB) {
	err := db.Gorm.AutoMigrate(j)
	if err != nil {
		panic(err)
	}
}

func (j *Journal) TableName() string { return tableName }

func (j *Journal) Upsert(db *btypes.DB, pc *btypes.ParamsContext) (pairs btypes.Pairs, err error) {
	var inserted bool
	inserted, err = pc.Tabler.Model().Upsert(db, pc.Tabler)
	if err != nil {
		return nil, err
	}

	if inserted {
		pairs.Add("msg", "添加成功")
	} else {
		pairs.Add("msg", "修改成功")
	}
	return
}

// Query 对于该表进行查询
// @params: 代表查询的参数
// return string: 代表该返回在Payload里的key
// return interface{}: 代表该返回key对应的结果
func (j *Journal) Query(db *btypes.DB, pc *btypes.ParamsContext) (pairs btypes.Pairs, err error) {
	var total int64
	var list []*Journal

	if pc.QueryParams == nil {
		panic("参数不能为空")
	}

	tx := db.Gorm.Begin()
	defer tx.Commit()

	tx = tx.Table(tableName)

	if pc.QueryParams.Conds == nil {
		// todo 还需对Conds重新设计
		tx = tx.Where(strings.Join(pc.QueryParams.Conds, ""))
	}
	// TODO: 所有total需根据是否是硬删除，如果是软删除，必须将deleted_at IS NULL
	if err := tx.Order(pc.QueryParams.Orderby).
		Offset(int(pc.QueryParams.Offset)).
		Limit(int(pc.QueryParams.Size)).
		Find(&list).Count(&total).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			tx.Rollback()
			panic(err)
		}
	}
	pairs.Add("total", total)
	pairs.Add(tableName, list)

	return
}

func (j *Journal) Delete(db *btypes.DB, pc *btypes.ParamsContext) (pairs btypes.Pairs, err error) {
	log.Printf("delete %#v", pc.Tabler)
	var n int64
	n, err = pc.Tabler.Model().SoftDelete(db, pc.Tabler)
	if err == nil {
		pairs.Add("msg", fmt.Sprintf("成功删除[%d]", n))
		return
	}
	return nil, err
}

func (j *Journal) Dispose(err error) (bool, error) {
	return false, err
}

func (j *Journal) FromRequest(rw json.RawMessage) btypes.Tabler {
	return btypes.FromRequestPayload(rw, j)
}
