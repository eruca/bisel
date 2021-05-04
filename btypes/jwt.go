package btypes

// WhiteList JWT的白名单，因为存在Upsert,所以还需要是插入或是更新
// 比如注册时不需要权限，而更新时需要权限
type WhiteList func(*ParamsContext) bool
