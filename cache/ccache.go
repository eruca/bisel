package cache

import (
	"fmt"
	"time"

	"github.com/bluele/gcache"
	"github.com/eruca/bisel/btypes"
	"github.com/karlseguin/ccache/v2"
)

const (
	expire    = 12 * time.Hour
	cacheSize = 1024
)

var _ btypes.Cacher = (*Cache)(nil)

type Cache struct {
	*ccache.LayeredCache
	gcache.Cache
	logger btypes.Logger
}

func New(logger btypes.Logger) *Cache {
	return &Cache{
		ccache.Layered(ccache.Configure()),
		gcache.New(cacheSize).ARC().Expiration(expire).Build(),
		logger,
	}
}

func (c *Cache) Size() string {
	return fmt.Sprintf("%d + %d", c.Cache.Len(false), c.LayeredCache.ItemCount())
}

func (c *Cache) SetBucket(tableName, hashKey string, value []byte) {
	c.LayeredCache.Set(tableName, hashKey, value, expire)
}

func (c *Cache) GetBucket(tableName, hashKey string) []byte {
	item := c.LayeredCache.Get(tableName, hashKey)
	if item == nil || item.Expired() {
		return nil
	}
	bin, ok := item.Value().([]byte)
	if !ok {
		c.logger.Errorf("%s:%s 存储的格式不是[]byte", tableName, hashKey)
		panic("存储格式不是[]byte")
	}
	return bin
}

func (c *Cache) ClearBuckets(tableNames ...string) {
	for _, tableName := range tableNames {
		c.LayeredCache.DeleteAll(tableName)
	}
}

func (c *Cache) Set(key, value interface{}) {
	err := c.Cache.Set(key, value)
	if err != nil {
		c.logger.Errorf("Set %v:%v failed: %v", key, value, err)
		panic("Cache Set(key, value) failed")
	}
}

func (c *Cache) Get(key interface{}) (interface{}, bool) {
	v, err := c.Cache.Get(key)
	if err != nil {
		return nil, false
	}
	return v, true
}

func (c *Cache) Remove(key interface{}) bool {
	return c.Cache.Remove(key)
}
