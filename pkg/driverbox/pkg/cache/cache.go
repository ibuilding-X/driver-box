// 缓存助手

package cache

import (
	"container/list"
	"sync"
	"time"
)

var defaultCache *ExpiringCache
var once = &sync.Once{}

// 缓存项结构体，包含值和过期时间
type cacheItem struct {
	key        string
	value      interface{}
	expiration time.Time
	element    *list.Element // 用于LRU淘汰策略
}

// 缓存结构体
type ExpiringCache struct {
	mu               sync.RWMutex
	items            map[string]*cacheItem
	ttl              time.Duration                       // 默认过期时间
	cleanupInterval  time.Duration                       // 清理间隔
	stopCleanup      chan struct{}                       // 停止清理的信号
	lruList          *list.List                          // LRU列表
	maxSize          int                                 // 最大缓存大小，0表示无限制
	evictionCallback func(key string, value interface{}) // 淘汰回调函数
}

func DefaultCache() *ExpiringCache {
	once.Do(func() {
		defaultCache = NewExpiringCache()
	})
	return defaultCache
}

// 选项函数，用于配置缓存
type Option func(*ExpiringCache)

// WithTTL 设置默认过期时间
func WithTTL(ttl time.Duration) Option {
	return func(c *ExpiringCache) {
		c.ttl = ttl
	}
}

// WithCleanupInterval 设置清理间隔
func WithCleanupInterval(interval time.Duration) Option {
	return func(c *ExpiringCache) {
		c.cleanupInterval = interval
	}
}

// WithMaxSize 设置最大缓存大小，用于LRU淘汰
func WithMaxSize(maxSize int) Option {
	return func(c *ExpiringCache) {
		c.maxSize = maxSize
	}
}

// WithEvictionCallback 设置淘汰回调函数
func WithEvictionCallback(callback func(key string, value interface{})) Option {
	return func(c *ExpiringCache) {
		c.evictionCallback = callback
	}
}

// 新建缓存实例
func NewExpiringCache(opts ...Option) *ExpiringCache {
	c := &ExpiringCache{
		items:           make(map[string]*cacheItem),
		ttl:             5 * time.Minute, // 默认5分钟过期
		cleanupInterval: 1 * time.Minute, // 默认1分钟清理一次
		stopCleanup:     make(chan struct{}),
		lruList:         list.New(),
		maxSize:         0, // 默认无大小限制
	}

	// 应用选项
	for _, opt := range opts {
		opt(c)
	}

	// 启动清理 goroutine
	go c.startCleanup()

	return c
}

// 设置缓存项，使用默认过期时间
func (c *ExpiringCache) Set(key string, value interface{}) {
	c.SetWithTTL(key, value, c.ttl)
}

// 设置缓存项，指定过期时间
func (c *ExpiringCache) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 如果键已存在，先移除旧的
	if item, exists := c.items[key]; exists {
		c.lruList.Remove(item.element)
		if c.evictionCallback != nil {
			c.evictionCallback(key, item.value)
		}
	}

	// 计算过期时间
	expiration := time.Now().Add(ttl)

	// 添加新项到LRU列表头部
	element := c.lruList.PushFront(key)

	// 保存缓存项
	c.items[key] = &cacheItem{
		key:        key,
		value:      value,
		expiration: expiration,
		element:    element,
	}

	// 如果设置了最大大小，检查是否需要淘汰
	if c.maxSize > 0 && c.lruList.Len() > c.maxSize {
		c.evictOldest()
	}
}

// 获取缓存项
func (c *ExpiringCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	item, exists := c.items[key]
	// 检查是否存在且未过期
	if !exists || time.Now().After(item.expiration) {
		c.mu.RUnlock()
		return nil, false
	}
	c.mu.RUnlock()

	// 更新LRU位置
	c.mu.Lock()
	c.lruList.MoveToFront(item.element)
	c.mu.Unlock()

	return item.value, true
}

// 删除缓存项
func (c *ExpiringCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if item, exists := c.items[key]; exists {
		c.lruList.Remove(item.element)
		delete(c.items, key)
		if c.evictionCallback != nil {
			c.evictionCallback(key, item.value)
		}
	}
}

// 清除所有缓存项
func (c *ExpiringCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, item := range c.items {
		if c.evictionCallback != nil {
			c.evictionCallback(item.key, item.value)
		}
	}
	c.items = make(map[string]*cacheItem)
	c.lruList.Init()
}

// 获取当前缓存大小
func (c *ExpiringCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// 启动清理过期项的goroutine
func (c *ExpiringCache) startCleanup() {
	ticker := time.NewTicker(c.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanupExpired()
		case <-c.stopCleanup:
			return
		}
	}
}

// 清理过期的缓存项
func (c *ExpiringCache) cleanupExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	// 遍历LRU列表，从尾部开始清理过期项
	for e := c.lruList.Back(); e != nil; {
		next := e.Prev()
		key := e.Value.(string)
		item := c.items[key]

		if now.After(item.expiration) {
			c.lruList.Remove(e)
			delete(c.items, key)
			if c.evictionCallback != nil {
				c.evictionCallback(key, item.value)
			}
		} else {
			// LRU列表尾部是最旧的，前面的不会比它更旧，所以可以退出循环
			break
		}

		e = next
	}
}

// 淘汰最旧的缓存项（LRU策略）
func (c *ExpiringCache) evictOldest() {
	// 获取最旧的项（列表尾部）
	oldest := c.lruList.Back()
	if oldest == nil {
		return
	}

	key := oldest.Value.(string)
	item := c.items[key]

	// 移除最旧的项
	c.lruList.Remove(oldest)
	delete(c.items, key)

	// 触发回调
	if c.evictionCallback != nil {
		c.evictionCallback(key, item.value)
	}
}

// 关闭缓存，停止清理goroutine
func (c *ExpiringCache) Close() {
	close(c.stopCleanup)
}
