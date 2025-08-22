// 插件接口

package plugin

import (
	"encoding/json"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"
)

// ToJSON 设备数据转 json
func (d DeviceData) ToJSON() string {
	b, _ := json.Marshal(d)
	return string(b)
}

// Plugin 驱动插件
type Plugin interface {
	// Initialize 初始化日志、配置、接收回调
	Initialize(logger *zap.Logger, c config.Config, ls *lua.LState)
	// Connector 连接器
	Connector(deviceId string) (connector Connector, err error)
	// Destroy 销毁驱动
	Destroy() error
}

// Connector 连接器
type Connector interface {
	Encode(deviceId string, mode EncodeMode, values ...PointData) (res interface{}, err error) // 编码，是否支持批量的读写操作，由各插件觉得
	Send(data interface{}) (err error)                                                         // 发送数据
	Release() (err error)                                                                      // 释放连接资源
}
