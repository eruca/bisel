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

var (
	_ Parameter = (*QueryParameter)(nil)
	_ Parameter = (*WriterParameter)(nil)
)

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

// ParamType 代表参数类型，作为辨别具体的写操作
type ParamType uint8

const (
	ParamInsert ParamType = iota
	ParamUpdate
	ParamDelete
)

func (pt ParamType) String() string {
	switch pt {
	case ParamInsert:
		return "INSERT"
	case ParamUpdate:
		return "UPDATE"
	case ParamDelete:
		return "DELETE"
	}
	panic("should not happened")
}

//*********** Params for fetch data ********************************
type QueryParameter struct {
	CheckJWT bool `json:"-"`
	QueryParam
}

func (qp *QueryParameter) JwtCheck() bool { return qp.CheckJWT }

type QueryParam struct {
	Conds   []string `json:"conds,omitempty"` // 限制条件
	Offset  uint64   `json:"offset,omitempty"`
	Size    int64    `json:"size,omitempty"` // 负数代表所有数据
	Orderby string   `json:"orderby,omitempty"`
	//! 需要客户端协调
	ForceUpdated bool `json:"force_updated,omitempty"` // 强制刷新，查询数据库
}

func (qp *QueryParam) String() string        { return "Query" }
func (qp *QueryParam) ReadForceUpdate() bool { return qp.ForceUpdated }
func (qp *QueryParam) Status() RequestStatus { return StatusRead }
func (qp *QueryParam) FromRawMessage(tabler Tabler, rm json.RawMessage) {
	err := json.Unmarshal(rm, qp)
	if err != nil {
		panic(err)
	}
	if qp.Orderby == "" {
		qp.Orderby = tabler.Orderby()
	}

	// 如果客户端未设置size,就使用服务端tabler的该表的默认设置
	// 如果size < 0, 则表示要求所有数据
	if qp.Size == 0 {
		qp.Size = int64(tabler.Size())
	}
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

	// 使用数组代替切片
	buf := [binary.MaxVarintLen64]byte{}
	n := binary.PutUvarint(buf[:], qp.Offset)
	wr.Write(buf[:n])

	n = binary.PutVarint(buf[:], qp.Size)
	wr.Write(buf[:n])

	wr.WriteString(qp.Orderby)

	err := wr.Flush()
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%X", hasher.Sum(nil))
}

func (qp *QueryParam) Call(c *Context, tabler Tabler) (Result, error) {
	return tabler.Query(c, tabler, qp, c.JwtSess)
}

// -------------------------WriterParam---------------------------------
type WriterParameter struct {
	ParamType `json:"-"`
	CheckJWT  bool `json:"-"`
	Tabler
}

func (wp *WriterParameter) FromRawMessage(tabler Tabler, rm json.RawMessage) {
	if len(rm) == 0 {
		panic("should have something in json.RawMessage")
	}
	err := json.Unmarshal(rm, tabler)
	if err != nil {
		panic(err)
	}
	wp.Tabler = tabler
}

func (wp *WriterParameter) Status() RequestStatus       { return StatusWrite }
func (wp *WriterParameter) BuildCacheKey(string) string { panic("no build key") }
func (wp *WriterParameter) JwtCheck() bool              { return wp.CheckJWT }
func (wp *WriterParameter) ReadForceUpdate() bool       { return false }

func (wp *WriterParameter) Call(c *Context, tabler Tabler) (Result, error) {
	switch wp.ParamType {
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
