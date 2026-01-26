package plugin

import "github.com/ibuilding-x/driver-box/driverbox/plugin"

const MirrorConnectionKey = "mirror_connection_key"

type encodeModel struct {
	deviceId string
	points   []plugin.PointData
	mode     plugin.EncodeMode
}

// rawDevice 原始设备
type rawDevice struct {
	deviceId  string
	pointName string
}
