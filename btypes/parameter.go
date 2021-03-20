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

// 默认的查询的大小
const DEFAULT_QUERY_SIZE = 20

// ParamType 代表参数类型，作为辨别ACID操作
type ParamType uint8

const (
	ParamQuery ParamType = iota
	ParamUpsert
	ParamDelete
)

type ParamsContext struct {
	ParamType
	*QueryParams
	Tabler
}

func ParamsContextFromJSON(tabler Tabler, pt ParamType, rw json.RawMessage) ParamsContext {
	switch pt {
	case ParamQuery:
		qp := &QueryParams{}
		err := json.Unmarshal(rw, qp)
		if err != nil {
			panic(err)
		}

		return ParamsContext{
			ParamType:   pt,
			QueryParams: qp,
		}
	case ParamUpsert, ParamDelete:
		tabler_new := tabler.New()
		err := json.Unmarshal(rw, tabler_new)
		if err != nil {
			panic(err)
		}

		return ParamsContext{
			ParamType: pt,
			Tabler:    tabler_new,
		}
	default:
		return ParamsContext{}
	}
}

func (pc *ParamsContext) init() {
	switch pc.ParamType {
	case ParamQuery:
		pc.QueryParams.init()
	}
}

func (pc *ParamsContext) CURD(db *DB, tabler Tabler) (Pairs, error) {
	switch pc.ParamType {
	case ParamQuery:
		return tabler.Query(db, pc)
	case ParamUpsert:
		return tabler.Upsert(db, pc)
	case ParamDelete:
		return tabler.Delete(db, pc)
	default:
		panic("should not happened")
	}
}

//*********** Params for fetch data ********************************
type QueryParams struct {
	Conds          []string `json:"conds,omitempty"` // 限制条件
	Offset         uint64   `json:"offset,omitempty"`
	Size           uint64   `json:"size,omitempty"`
	Orderby        string   `json:"orderby,omitempty"`
	ReforceUpdated bool     `json:"reforce_updated,omitempty"` // 强制刷新，查询数据库
	Hash           string   `json:"hash,omitempty"`
}

func (qp *QueryParams) Type() ParamType {
	return ParamQuery
}

// Init 中的list目的是获取外部指针，接收内部产生的数据作为返回
func (qp *QueryParams) init() {
	if qp.Conds != nil {
		sort.Strings(qp.Conds)
	}
	if qp.Size == 0 {
		qp.Size = DEFAULT_QUERY_SIZE
	}
	if qp.Orderby == "" {
		qp.Orderby = "id desc"
	}
}

// BuildCacheKey是按上面的结构体顺序输入，计算md5
// 因为Hash是作为服务器返回给客户的数据，作为重复查询的话可以使用缓存的目的
// todo: sort.Strings(qp.Conds) 提前到初始化时
func (qp *QueryParams) BuildCacheKey(req *Request) string {
	sort.Strings(qp.Conds)
	hasher := md5.New()
	wr := bufio.NewWriter(hasher)
	// 查询的类型必须放进去，要不然不同的查询都是同一结果
	wr.WriteString(req.Type)

	if len(qp.Conds) > 0 {
		wr.WriteString(strings.Join(qp.Conds, ""))
	}

	buf := make([]byte, binary.MaxVarintLen64)
	binary.PutUvarint(buf, qp.Offset)
	wr.Write(buf)

	// 不重复使用buf，
	//? 假设len(buf) == 4, 如果第一个是4个长度，第二个是2个长度，就有问题
	buf = make([]byte, binary.MaxVarintLen64)
	binary.PutUvarint(buf, qp.Size)
	wr.Write(buf)

	wr.WriteString(qp.Orderby)

	err := wr.Flush()
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%X", hasher.Sum(nil))
}
