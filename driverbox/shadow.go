package driverbox

import (
	shadow0 "github.com/ibuilding-x/driver-box/driverbox/shadow"
	"github.com/ibuilding-x/driver-box/internal/shadow"
)

// Shadow 获取设备影子服务实例
// 设备影子服务用于维护设备状态，提供状态同步和监控功能
func Shadow() shadow0.DeviceShadow {
	return shadow.Shadow()
}
