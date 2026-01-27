package driverbox

import "github.com/ibuilding-x/driver-box/internal/cache"

// CoreCache 获取核心缓存实例
// 提供对系统核心缓存的访问，用于存储和检索运行时数据
// 核心缓存包含设备配置、点位信息、模型定义等关键数据
// 返回值:
//   - cache.CoreCache: 核心缓存实例，可通过该实例访问设备、点位、模型等信息
//
// 使用示例:
//
//	cache := driverbox.CoreCache()
//	devices := cache.Devices()
//	device, exists := cache.GetDevice("device001")
func CoreCache() cache.CoreCache {
	return cache.Get()
}
