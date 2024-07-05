package discover

import "github.com/ibuilding-x/driver-box/driverbox/config"

type DeviceDiscover struct {
	ModelKey      string        `json:"modelKey"` //模型Key
	Device        config.Device `json:"device"`
	ProtocolName  string        `json:"protocolName"`
	ConnectionKey string        `json:"connectionKey"`
}
