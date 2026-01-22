package driverbox

import (
	shadow0 "github.com/ibuilding-x/driver-box/driverbox/shadow"
	"github.com/ibuilding-x/driver-box/internal/shadow"
)

// Shadow 获取设备影子服务实例
// 设备影子服务用于维护设备状态，提供状态同步和监控功能
// 设备影子服务提供以下功能:
// 1. 设备状态持久化
// 2. 在离线状态监控
// 3. 点位值缓存
// 4. 控制指令存储
//
// 返回值:
//   - shadow0.DeviceShadow: 设备影子服务实例
//
// 使用示例:
//
//	shadowService := driverbox.Shadow()
//	shadowService.AddDevice("device001", "temp_sensor")
//	value, err := shadowService.GetDevicePoint("device001", "temperature")
//	if err != nil {
//	    driverbox.Log().Error("Get device point failed", zap.Error(err))
//	}
func Shadow() shadow0.DeviceShadow {
	return shadow.Shadow()
}