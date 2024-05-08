package library

import "github.com/ibuilding-x/driver-box/driverbox/plugin"

// 设备驱动编码请求
type DeviceEncodeRequest struct {
	DeviceId string // 设备ID
	Mode     plugin.EncodeMode
	Points   []plugin.PointData
}

// 设备驱动编码结果
type DeviceEncodeResult struct {
	Points []plugin.PointData
	Error  error
}

// 设备驱动解码请求
type DeviceDecodeRequest struct {
	DeviceId string             `json:"id"` // 设备ID
	Points   []plugin.PointData `json:"points"`
}

// 设备驱动解码结果
type DeviceDecodeResult struct {
	//解码结果
	Points []plugin.PointData `json:"points"`
	//解码错误信息
	Error error `json:"error"`
}
