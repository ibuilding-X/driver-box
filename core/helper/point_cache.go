// 点位缓存

package helper

import (
	"sync"
	"time"
)

var PointCache PointCacheClient // 点位缓存客户端

// PointCacheClient 点位缓存客户端
type PointCacheClient interface {
	SetDefaultTTL(ttl time.Duration)                                                  // 设置默认 TTL
	Get(deviceName string, pointName string) (value interface{})                      // 获取缓存点位信息
	Set(deviceName string, pointName string, value interface{}, ttl ...time.Duration) // 设置缓存点位信息
	CleanAll()                                                                        // 清空所有缓存
}

type pointCache struct {
	ttl time.Duration
	l   *sync.Mutex
	m   map[string]cacheData
}

// cacheData 缓存数据
type cacheData struct {
	value     interface{}
	expiredAt time.Time
}

// InitPointCache 初始化点位缓存客户端
func InitPointCache(ttl time.Duration) {
	PointCache = &pointCache{
		ttl: ttl,
		l:   &sync.Mutex{},
		m:   make(map[string]cacheData),
	}
}

func (pc *pointCache) SetDefaultTTL(ttl time.Duration) {
	pc.l.Lock()
	defer pc.l.Unlock()
	pc.ttl = ttl
}

func (pc *pointCache) Get(deviceName string, pointName string) (value interface{}) {
	key := pc.genKey(deviceName, pointName)

	pc.l.Lock()
	defer pc.l.Unlock()

	data, ok := pc.m[key]
	if ok && data.expiredAt.After(time.Now()) {
		return data.value
	}

	return nil
}

func (pc *pointCache) Set(deviceName string, pointName string, value interface{}, ttl ...time.Duration) {
	key := pc.genKey(deviceName, pointName)
	expiredAt := time.Now().Add(pc.ttl)
	if len(ttl) > 0 {
		expiredAt = time.Now().Add(ttl[0])
	}

	pc.l.Lock()
	defer pc.l.Unlock()

	pc.m[key] = cacheData{
		value:     value,
		expiredAt: expiredAt,
	}
}

func (pc *pointCache) CleanAll() {
	pc.l.Lock()
	defer pc.l.Unlock()

	pc.m = make(map[string]cacheData)
}

func (pc *pointCache) genKey(deviceName, pointName string) string {
	return deviceName + "_" + pointName
}
