package manager

import "github.com/eruca/bisel/btypes"

// 否则默认使用responseType作为ConfigResponseType
func defaultResponseType(reqType string, successed bool) string {
	if successed {
		return reqType + "_success"
	}
	return reqType + "_failure"
}

// ManagerConfig对一些参数进行默认配置
type ManagerConfig struct {
	btypes.ConfigResponseType
	DefaultQuerySize int // 20
}

func (mc *ManagerConfig) init() {
	if mc.ConfigResponseType == nil {
		mc.ConfigResponseType = defaultResponseType
	}

	if mc.DefaultQuerySize == 0 {
		mc.DefaultQuerySize = btypes.DEFAULT_QUERY_SIZE
	}
}
