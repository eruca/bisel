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

// ParamType 代表参数类型，作为辨别CRUD操作
type ParamType uint8

const (
	ParamQuery ParamType = iota
	ParamUpsert
	ParamDelete
	ParamLogin
)

func (p ParamType) String() string {
	switch p {
	case ParamQuery:
		return "ParamQuery"
	case ParamUpsert:
		return "ParamUpsert"
	case ParamDelete:
		return "ParamDelete"
	case ParamLogin:
		return "ParamLogin"
	}
	return ""
}

type ParamsContext struct {
	// Param的类型
	ParamType
	// Param具体的值，
	// *QueryParams 保存查询时的参数
	*QueryParams
	// Tabler 保存Upsert/Delete的参数
	// 因为这是按照这个对象增加/修改
	// 或对象的id, version进行删除
	Tabler
}

func ParamsContextForConnectter(fullSize bool) ParamsContext {
	qp := &QueryParams{}
	qp.init(fullSize)
	return ParamsContext{ParamType: ParamQuery, QueryParams: qp}
}

func ParamsContextFromJSON(tabler Tabler, pt ParamType, rw json.RawMessage) (pc ParamsContext) {
	switch pt {
	case ParamQuery:
		pc.ParamType = pt
		pc.QueryParams = &QueryParams{}

		if rw != nil {
			err := json.Unmarshal(rw, &pc.QueryParams)
			if err != nil {
				panic(err)
			}
		}
	case ParamUpsert, ParamDelete, ParamLogin:
		pc.ParamType = pt
		if rw == nil {
			// todo: 如果发送的信息，没有信息体，是否需要panic
			panic(fmt.Sprintf("%s is nil", pt))
		}

		pc.Tabler = tabler
		err := json.Unmarshal(rw, pc.Tabler)
		if err != nil {
			panic(err)
		}
	default:
		panic("never happened")
	}

	pc.init()
	return pc
}

func (pc *ParamsContext) init() {
	switch pc.ParamType {
	case ParamQuery:
		pc.QueryParams.init(false)
	}
}

func (pc *ParamsContext) Assemble(value fmt.Stringer) PairStringer {
	switch pc.ParamType {
	case ParamQuery:
		return PairStringer{Key: "QUERY", Value: value}
	case ParamUpsert:
		return PairStringer{Key: "UPSERT", Value: value}
	case ParamDelete:
		return PairStringer{Key: "DELETE", Value: value}
	case ParamLogin:
		return PairStringer{Key: "LOGIN", Value: value}
	default:
		panic("never happen")
	}
}

// ParamsContext 针对不同的ParamType 采取相应的处理
func (pc *ParamsContext) Do(injected Injected, tabler Tabler, jwtSession Defaulter) (Result, error) {
	switch pc.ParamType {
	case ParamQuery:
		// 应为客户端传送过来的数据不会序列化为Tabler，所以使用注入tabler
		return tabler.Query(injected.DB, pc, jwtSession)

	case ParamUpsert:
		// pc.Tabler代表从客户端过来序列化后的数据
		return pc.Tabler.Upsert(injected.DB, pc, jwtSession)

	case ParamDelete:
		// 同ParamUpsert
		return pc.Tabler.Delete(injected.DB, pc, jwtSession)

	case ParamLogin:
		loginTabler, ok := pc.Tabler.(LoginTabler)
		if !ok {
			panic(fmt.Sprintf("%s 没有实现 LoginTabler", pc.TableName()))
		}
		return loginJWT(injected, loginTabler, jwtSession)

	default:
		panic("should not happened")
	}
}

//*********** Params for fetch data ********************************
type QueryParams struct {
	Conds          []string `json:"conds,omitempty"` // 限制条件
	Offset         uint64   `json:"offset,omitempty"`
	Size           int64    `json:"size,omitempty"`
	Orderby        string   `json:"orderby,omitempty"`
	ReforceUpdated bool     `json:"reforce_updated,omitempty"` // 强制刷新，查询数据库
}

func (qp *QueryParams) Type() ParamType {
	return ParamQuery
}

// Init 中的list目的是获取外部指针，接收内部产生的数据作为返回
func (qp *QueryParams) init(fullSize bool) {
	if qp.Conds != nil {
		sort.Strings(qp.Conds)
	}

	if fullSize {
		qp.Size = -1
	} else {
		if qp.Size == 0 {
			qp.Size = DEFAULT_QUERY_SIZE
		}
	}

	if qp.Orderby == "" {
		qp.Orderby = "updated_at desc"
	}
}

// BuildCacheKey是按上面的结构体顺序输入，计算md5
// 因为Hash是作为服务器返回给客户的数据，作为重复查询的话可以使用缓存的目的
// todo: sort.Strings(qp.Conds) 提前到初始化时
func (qp *QueryParams) BuildCacheKey(reqType string) string {
	sort.Strings(qp.Conds)
	hasher := md5.New()
	wr := bufio.NewWriter(hasher)
	// 查询的类型必须放进去，要不然不同的查询都是同一结果
	wr.WriteString(reqType)

	if len(qp.Conds) > 0 {
		wr.WriteString(strings.Join(qp.Conds, ""))
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
