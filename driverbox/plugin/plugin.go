// 插件接口

package plugin

import (
	"encoding/json"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"
)

// 触发 ExportTo 的类型
type ExportType string

const (
	ReadMode       EncodeMode = "read"           // 读模式
	WriteMode      EncodeMode = "write"          // 写模式
	RealTimeExport ExportType = "realTimeExport" //实时上报
	TimerExport    ExportType = "timerExport"    //周期性上报
)

// EncodeMode 编码模式
type EncodeMode string

// H map[string]interface{} 别名
type H map[string]interface{}

// OnReceiveHandler 接收数据回调函数
type OnReceiveHandler func(plugin Plugin, raw interface{}) (result interface{}, err error)

// PointData 点位数据
type PointData struct {
	PointName string      `json:"name"`  // 点位名称
	Value     interface{} `json:"value"` // 点位值
}

// DeviceData 设备数据
type DeviceData struct {
	SN         string      `json:"sn"`
	Values     []PointData `json:"values"`
	ExportType ExportType  //上报类型，底层的变化上报和实时上报等同于RealTimeExport
}

// ToJSON 设备数据转 json
func (d DeviceData) ToJSON() string {
	b, _ := json.Marshal(d)
	return string(b)
}

// Plugin 驱动插件
type Plugin interface {
	// Initialize 初始化日志、配置、接收回调
	Initialize(logger *zap.Logger, c config.Config, handler OnReceiveHandler, ls *lua.LState) (err error)
	// ProtocolAdapter 协议适配器
	ProtocolAdapter() ProtocolAdapter
	// Connector 连接器
	Connector(deviceSn, pointName string) (connector Connector, err error)
	// Destroy 销毁驱动
	Destroy() error
}

// Connector 连接器
type Connector interface {
	Send(data interface{}) (err error) // 发送数据
	Release() (err error)              // 释放连接资源
}

// ProtocolAdapter 协议适配器
// 点位数据 <=> 协议数据
type ProtocolAdapter interface {
	Encode(deviceSn string, mode EncodeMode, value PointData) (res interface{}, err error) // 编码
	Decode(raw interface{}) (res []DeviceData, err error)                                  // 解码
}
