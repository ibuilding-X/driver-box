package config

// DeviceConfig res/driver目录下的config.json
type DeviceConfig struct {
	// 设备模型
	DeviceModels []DeviceModel `json:"deviceModels" validate:""`
	// 连接配置
	Connections map[string]interface{} `json:"connections" validate:""`
	// 协议名称（通过协议名称区分连接模式：客户端、服务端）
	PluginName string `json:"protocolName" validate:"required"`
}

// DeviceModel 设备模型
type DeviceModel struct {
	Model
	// 设备列表
	Devices []Device `json:"devices" validate:"required"`
}
