package discover

import "github.com/ibuilding-x/driver-box/driverbox/pkg/config"

type DeviceDiscover struct {
	ModelName string        `json:"modelName"` //模型名称后缀
	ModelKey  string        `json:"modelKey"`  //模型Key
	Device    config.Device `json:"device"`
	//模型自定义属性
	Model         map[string]map[string]any `json:"model"`
	ProtocolName  string                    `json:"protocolName"`
	ConnectionKey string                    `json:"connectionKey"`
}
