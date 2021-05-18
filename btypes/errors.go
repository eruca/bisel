package btypes

import "errors"

var (
	ErrOptimisticLock                      = errors.New("乐观锁错误: 数据已经被修改，请刷新后重新请求")
	ErrAccountNotExistOrPasswordNotCorrect = errors.New("账号不存在或密码错误")
	ErrInvalidToken                        = errors.New("无效的token")
	ErrTokenExpired                        = errors.New("token过期")
	ErrTableIsOnEditting                   = errors.New("目前该条数据正在编辑中")
	ErrTableIsOffEditting                  = errors.New("目前该条数据已经不在编辑了")

	ErrStringUniqueConstrait = "unique constraint"
)
