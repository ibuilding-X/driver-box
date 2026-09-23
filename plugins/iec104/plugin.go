package iec104

import (
	"github.com/ibuilding-x/driver-box/v2/driverbox"
	"github.com/ibuilding-x/driver-box/v2/plugins/iec104/internal"
)

// EnablePlugin 注册 IEC104 主站；配置与设备加载由 driverbox.Start 统一完成。
func EnablePlugin() { driverbox.EnablePlugin(internal.ProtocolName, new(internal.Plugin)) }
