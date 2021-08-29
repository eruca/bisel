package btypes

// ConnectionType 代表连接类型
// 1. http请求 2.websocket请求
type ConnectionType uint8

const (
	HTTP ConnectionType = iota
	WEBSOCKET
)

// ConfigResponseType 让使用者可以定制返回的Type结果
type ConfigResponseType func(msg string, success bool) string
