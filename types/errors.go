package types

import "errors"

var (
	ErrOptimisticLock = errors.New("乐观锁错误: 数据已经被修改，请刷新后重新请求")
	ErrGroup          = NewErrorGroup()
)

type ErrorGroup []PairString

func NewErrorGroup() ErrorGroup {
	return []PairString{
		{Key: "UNIQUE constraint failed:", Value: "违反唯一限制: %s"},
	}
}
