package driverbox

import "github.com/ibuilding-x/driver-box/internal/cache"

// CoreCache 获取核心缓存实例
// 提供对系统核心缓存的访问，用于存储和检索运行时数据
// 返回值:
//   - cache.CoreCache: 核心缓存实例
//
// 使用示例:
//
//	cache := driverbox.CoreCache()
//	devices := cache.Devices()
func CoreCache() cache.CoreCache {
	return cache.Get()
}
