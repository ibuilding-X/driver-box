package driverbox

import (
	shadow0 "github.com/ibuilding-x/driver-box/driverbox/shadow"
	"github.com/ibuilding-x/driver-box/internal/shadow"
)

// Shadow 获取设备影子服务实例
// 设备影子服务用于维护设备状态，提供状态同步和监控功能
// 设备影子服务提供以下功能:
// 1. 设备状态持久化 - 保存设备最新状态
// 2. 在离线状态监控 - 跟踪设备连接状态
// 3. 点位值缓存 - 缓存设备点位的最新值
// 4. 控制指令存储 - 记录下发给设备的控制指令
//
// 返回值:
//   - shadow0.DeviceShadow: 设备影子服务实例，提供设备状态管理的完整接口
//
// 使用示例:
//
//	shadowService := driverbox.Shadow()
//	shadowService.AddDevice("device001", "temp_sensor")
//	value, err := shadowService.GetDevicePoint("device001", "temperature")
//	if err != nil {
//	    driverbox.Log().Error("Get device point failed", zap.Error(err))
//	}
//	status, err := shadowService.IsOnline("device001")
//	if err != nil {
//	    driverbox.Log().Error("Check device online status failed", zap.Error(err))
//	}
func Shadow() shadow0.DeviceShadow {
	return shadow.Shadow()
}
