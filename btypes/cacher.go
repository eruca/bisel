package btypes

import (
	"os"
	"sync"

	lru "github.com/hashicorp/golang-lru"
)

const CACHESIZE = 512

var _ Cacher = (*ARC)(nil)

// Cacher 目标是将客户端请求Cache化
// 每个请求都不一致，所以对请求做hash, 保证请求一致时可以用缓存
// 如果对表进行了Update/Delete/Insert，将该表所有缓存删除
type Cacher interface {
	Get(string) ([]byte, bool)
	Set(string, string, []byte)
	Clear(...string)
}

type ARC struct {
	cacher  *lru.ARCCache
	records map[string][]string
	logger  Logger
	sync.Mutex
}

func NewCacher(logger Logger) *ARC {
	cacher, err := lru.NewARC(CACHESIZE)
	if err != nil {
		logger.Errorf("lru.NewARC(%d) 发生错误: %s", CACHESIZE, err.Error())
		os.Exit(1)
	}
	return &ARC{
		cacher:  cacher,
		records: map[string][]string{},
	}
}

func (arc *ARC) Get(key string) ([]byte, bool) {
	if v, ok := arc.cacher.Get(key); ok {
		if data, ok := v.([]byte); ok {
			return data, true
		} else {
			arc.logger.Errorf("插入%q时不是[]byte类型数据", key)
			panic("插入不是[]byte类型")
		}
	}
	return nil, false
}

func (arc *ARC) Set(tableName, key string, value []byte) {
	arc.Lock()
	defer arc.Unlock()

	if slices, ok := arc.records[tableName]; ok {
		arc.records[tableName] = append(slices, key)
	} else {
		arc.records[tableName] = []string{key}
	}
	//! 这个是原子操作，可是需要为了和map同步，就一起上锁了
	arc.cacher.Add(key, value)
}

func (arc *ARC) Clear(tableNames ...string) {
	arc.Lock()
	defer arc.Unlock()

	for _, tableName := range tableNames {
		if keys, ok := arc.records[tableName]; ok {
			for _, key := range keys {
				arc.cacher.Remove(key)
			}
		}
		// 同时删除records里的该tableName
		delete(arc.records, tableName)
	}
}
