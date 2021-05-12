package btypes

import (
	"fmt"
)

// Pair 代表Key:Value
type Pair struct {
	Key   string      `json:"key,omitempty"`
	Value interface{} `json:"value,omitempty"`
}

type Pairs []Pair

func (ps *Pairs) Add(key string, value interface{}) {
	*ps = append(*ps, Pair{Key: key, Value: value})
}

// ValueString 直接将string实现fmt.Stringer
type ValueString string

func (vs ValueString) String() string {
	return string(vs)
}

// PairString ...
// type PairString struct {
// 	Key   string      `json:"key,omitempty"`
// 	Value ValueString `json:"value,omitempty"`
// }

// PairStringer 调试时可以比较好控制输出
type PairStringer struct {
	Key   string       `json:"key,omitempty"`
	Value fmt.Stringer `json:"value,omitempty"`
}
