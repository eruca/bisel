package users

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/eruca/bisel/bin/models"
	"github.com/eruca/bisel/btypes"
	"github.com/eruca/bisel/middlewares"
)

var (
	_         btypes.Tabler  = (*User)(nil)
	_         btypes.Loginer = (*User)(nil)
	tableName string         = "users"
)

type User struct {
	btypes.GormModel
	Account  string `json:"account,omitempty"`
	Password string `json:"password,omitempty"`
	Name     string `json:"name,omitempty"`
	Ismale   bool   `json:"ismale,omitempty"`
	Age      uint8  `json:"age,omitempty"`
	Role     uint64 `json:"role,omitempty"`
}

func (j *User) GetAccount() btypes.PairStringer {
	return btypes.PairStringer{
		Key: "account", Value: btypes.ValueString(j.Account),
	}
}

func (j *User) GetPassword() btypes.PairStringer {
	return btypes.PairStringer{Key: "password", Value: btypes.ValueString(j.Password)}
}

func (j *User) New() btypes.Tabler {
	return &User{}
}

func (j *User) Model() *btypes.GormModel {
	return &j.GormModel
}

func (j *User) Register(handlers map[string]btypes.ContextConfig) {
	// 这里的j实际上是Manager.New时传入的对象
	// 这个对象应该一直都是空值，只是作为调用
	handlers[tableName+"/query"] = btypes.QueryHandler(j, middlewares.TimeElapsed, models.JwtAuth, middlewares.UseCache)
	handlers[tableName+"/upsert"] = btypes.UpsertHandler(j, middlewares.TimeElapsed, models.JwtAuth)
	handlers[tableName+"/delete"] = btypes.DeleteHandler(j, middlewares.TimeElapsed, models.JwtAuth)
	handlers[tableName+"/login"] = btypes.LoginHandler(j, &models.JwtSession{}, middlewares.TimeElapsed)
}

func (j *User) MustAutoMigrate(db *btypes.DB) {
	err := db.Gorm.AutoMigrate(j)
	if err != nil {
		panic(err)
	}
}

func (j *User) TableName() string { return tableName }

func (j *User) Upsert(db *btypes.DB, pc *btypes.ParamsContext, jwtSession btypes.Defaulter) (pairs btypes.Pairs, err error) {
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
func (j *User) Query(db *btypes.DB, pc *btypes.ParamsContext, jwtSession btypes.Defaulter) (pairs btypes.Pairs, err error) {
	var total int64
	var list []*User

	if sess, ok := jwtSession.(*models.JwtSession); ok {
		log.Printf("jwtSession: %#v", sess)
	} else {
		log.Println("not ok")
	}

	if pc.QueryParams == nil {
		panic("参数不能为空")
	}

	btypes.QueryAssist(db.Gorm, j, pc, &total, &list, "password")
	pairs.Add("total", total)
	pairs.Add(tableName, list)

	return
}

func (j *User) Delete(db *btypes.DB, pc *btypes.ParamsContext, jwtSession btypes.Defaulter) (pairs btypes.Pairs, err error) {
	log.Printf("delete %#v", pc.Tabler)
	var n int64
	n, err = pc.Tabler.Model().SoftDelete(db, pc.Tabler)
	if err == nil {
		pairs.Add("msg", fmt.Sprintf("成功删除[%d]", n))
		return
	}
	return nil, err
}

func (j *User) Dispose(err error) (bool, error) {
	return false, err
}

func (j *User) FromRequest(rw json.RawMessage) btypes.Tabler {
	return btypes.FromRequestPayload(rw, j)
}
