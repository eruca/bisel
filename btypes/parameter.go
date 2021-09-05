package btypes

import (
	"bufio"
	"crypto/md5"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

var _ Parameter = (*ParamContext)(nil)

type RequestStatus uint8

const (
	// 对cache没有影响
	StatusNoop RequestStatus = iota
	// 读操作，会产生缓存
	StatusRead
	// 写操作，会清缓存
	StatusWrite
)

type Parameter interface {
	String() string
	FromRawMessage(Tabler, json.RawMessage)
	JwtCheck() bool
	Status() RequestStatus
	ReadForceUpdate() bool
	BuildCacheKey(string) string
	Call(*Context, Tabler) (Result, error)
}

// ParamType 代表参数类型，作为辨别CRUD操作
type ParamType uint8

const (
	ParamQuery ParamType = iota
	ParamInsert
	ParamUpdate
	ParamDelete
)

func (pt ParamType) String() string {
	switch pt {
	case ParamQuery:
		return "QUERY"
	case ParamInsert:
		return "INSERT"
	case ParamUpdate:
		return "UPDATE"
	case ParamDelete:
		return "DELETE"
	}
	panic("should not happened")
}

type ParamContext struct {
	// Param的类型
	ParamType
	// QueryParam
	*QueryParam
	Tabler
}

func (pc *ParamContext) FromRawMessage(tabler Tabler, rm json.RawMessage) {
	if len(rm) == 0 {
		panic("should have something in json.RawMessage")
	}

	switch pc.ParamType {
	case ParamQuery:
		qp := &QueryParam{}
		err := json.Unmarshal(rm, qp)
		if err != nil {
			panic(err)
		}
		if qp.Orderby == "" {
			qp.Orderby = tabler.Orderby()
		}

		if qp.Size == 0 {
			qp.Size = int64(tabler.Size())
		}
		pc.QueryParam = qp

	case ParamInsert, ParamUpdate, ParamDelete:
		err := json.Unmarshal(rm, tabler)
		if err != nil {
			panic(err)
		}
		pc.Tabler = tabler
	}
}

func (pc *ParamContext) Status() RequestStatus {
	switch pc.ParamType {
	case ParamQuery:
		return StatusRead
	case ParamInsert, ParamUpdate, ParamDelete:
		return StatusWrite
	default:
		panic("should not happened")
	}
}

func (pc *ParamContext) BuildCacheKey(reqType string) string {
	switch pc.ParamType {
	case ParamInsert, ParamUpdate, ParamDelete:
		return ""
	case ParamQuery:
		if pc.QueryParam == nil {
			panic("should not be nil")
		}
		return pc.QueryParam.BuildCacheKey(reqType)
	default:
		panic("Should not happen")
	}
}

func (*ParamContext) JwtCheck() bool { return true }

func (pc *ParamContext) Call(c *Context, tabler Tabler) (Result, error) {
	switch pc.ParamType {
	case ParamQuery:
		return tabler.Query(c, tabler, pc.QueryParam, c.JwtSess)
	case ParamInsert:
		return tabler.Insert(c, tabler, c.JwtSess)
	case ParamUpdate:
		return tabler.Update(c, tabler, c.JwtSess)
	case ParamDelete:
		return tabler.Delete(c, tabler, c.JwtSess)
	default:
		panic("should not happened")
	}
}

//*********** Params for fetch data ********************************
type QueryParam struct {
	Conds   []string `json:"conds,omitempty"` // 限制条件
	Offset  uint64   `json:"offset,omitempty"`
	Size    int64    `json:"size,omitempty"`
	Orderby string   `json:"orderby,omitempty"`
	//! 需要客户端协调
	ForceUpdated bool `json:"force_updated,omitempty"` // 强制刷新，查询数据库
}

// BuildCacheKey是按上面的结构体顺序输入，计算md5
// 因为Hash是作为服务器返回给客户的数据，作为重复查询的话可以使用缓存的目的
// todo: sort.Strings(qp.Conds) 提前到初始化时
func (qp *QueryParam) BuildCacheKey(reqType string) string {
	hasher := md5.New()
	wr := bufio.NewWriter(hasher)
	// 查询的类型必须放进去，要不然不同的查询都是同一结果
	wr.WriteString(reqType)

	if len(qp.Conds) > 0 {
		conds := make([]string, len(qp.Conds))
		copy(conds, qp.Conds)

		sort.Strings(conds)
		wr.WriteString(strings.Join(conds, ""))
	}

	buf := make([]byte, binary.MaxVarintLen64)
	binary.PutUvarint(buf, qp.Offset)
	wr.Write(buf)

	// 不重复使用buf，重置buf
	//? 假设len(buf) == 4, 如果第一个是4个长度，第二个是2个长度，就有问题
	// buf = make([]byte, binary.MaxVarintLen64)
	for i := 0; i < binary.MaxVarintLen64; i++ {
		buf[i] = 0
	}
	binary.PutVarint(buf, qp.Size)
	wr.Write(buf)

	wr.WriteString(qp.Orderby)

	err := wr.Flush()
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%X", hasher.Sum(nil))
}

func (pc *QueryParam) ReadForceUpdate() bool {
	return pc.ForceUpdated
}
