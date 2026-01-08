package mirror

import "github.com/ibuilding-x/driver-box/pkg/driverbox/plugin"

type EncodeModel struct {
	deviceId string
	points   []plugin.PointData
	mode     plugin.EncodeMode
}

// Device 原始设备
type Device struct {
	deviceId  string
	pointName string
}
