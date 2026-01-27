package config

// 网关元数据
type Metadata struct {
	// SerialNo 网关在云端定义的SN号，用于唯一标识网关设备
	SerialNo string `json:"serialNo"`

	//产品型号，标识网关的硬件型号和规格
	//示例值:
	// - "GW1000": 基础型号，支持基础协议和功能
	// - "GW2000": 高级型号，支持所有协议和扩展功能
	//注意:
	// 1. 型号决定了网关支持的功能和性能参数
	// 2. 不同型号可能有不同的固件版本要求
	Model string `json:"model"`

	//厂商名称，标识网关的生产厂商
	//示例值: "Huawei", "ZTE", "Cisco"
	//注意:
	// 1. 厂商名称应与官方注册名称一致
	// 2. 用于区分不同厂商的设备兼容性和支持服务
	Vendor string `json:"vendor"`

	// 网关硬件唯一编号
	HardwareSN string `json:"hardwareSn"`

	// 集成电路卡识别码，即 SIM 卡卡号
	ICCID string `json:"iccid"`

	//软件版本号，例如："1.0.0"
	SoftwareVersion string `json:"softwareVersion"`
}
