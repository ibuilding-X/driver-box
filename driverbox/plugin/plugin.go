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
	Initialize(logger *zap.Logger, c config.Config, ls *lua.LState) (err error)
	// Connector 连接器
	Connector(deviceSn, pointName string) (connector Connector, err error)
	// Destroy 销毁驱动
	Destroy() error
}

// Connector 连接器
type Connector interface {
	// ProtocolAdapter 协议适配器
	ProtocolAdapter() ProtocolAdapter
	Send(data interface{}) (err error) // 发送数据
	Release() (err error)              // 释放连接资源
}

// ProtocolAdapter 协议适配器
// 点位数据 <=> 协议数据
type ProtocolAdapter interface {
	Encode(deviceSn string, mode EncodeMode, values ...PointData) (res interface{}, err error) // 编码，是否支持批量的读写操作，由各插件觉得
	Decode(raw interface{}) (res []DeviceData, err error)                                      // 解码
}
