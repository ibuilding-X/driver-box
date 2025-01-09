package discover

import "github.com/ibuilding-x/driver-box/driverbox/config"

type DeviceDiscover struct {
	ModelKey string        `json:"modelKey"` //模型Key
	Device   config.Device `json:"device"`
	//模型自定义属性
	Model         map[string]map[string]any `json:"modelName"`
	ProtocolName  string                    `json:"protocolName"`
	ConnectionKey string                    `json:"connectionKey"`
}
