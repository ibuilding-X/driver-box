package config

// Device 设备
type Device struct {
	// 设备 ID
	ID string `json:"id" validate:"required"`
	// 模型名称
	ModelName string `json:"-" validate:"-"`
	// 设备描述
	Description string `json:"description" validate:"required"`
	// 设备离线阈值，超过该时长没有收到数据视为离线
	//Ttl string `json:"ttl"`

	//设备标签
	//Tags []string `json:"tags"`
	// 连接 Key
	ConnectionKey string `json:"connectionKey" validate:"required"`
	// 协议参数
	Properties map[string]string `json:"properties" validate:"-"`

	//设备层驱动的引用
	DriverKey string `json:"driverKey"`

	//设备对应的协议
	PluginName string `json:"-" validate:"-"`
}
