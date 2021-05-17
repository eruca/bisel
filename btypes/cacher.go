package btypes

// Cacher 目标是将客户端请求Cache化
// 每个请求都不一致，所以对请求做hash, 保证请求一致时可以用缓存
// 如果对表进行了Update/Delete/Insert，将该表所有缓存删除
type Cacher interface {
	Get(interface{}) (interface{}, bool)
	Set(interface{}, interface{})
	GetBucket(string, string) []byte
	SetBucket(string, string, []byte)
	ClearBuckets(...string)
}
