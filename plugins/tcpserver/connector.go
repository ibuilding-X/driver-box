package tcpserver

import (
	"github.com/ibuilding-x/driver-box/v2/driverbox/plugin"
)

// Connector 是 TCP Server 插件在 plugin.Connector 基础上提供的可选扩展接口。
// 使用方在已经拿到 plugin.Connector 的场景下，可通过类型断言访问这些能力：
//
//	var conn plugin.Connector
//	if tcpConn, ok := conn.(tcpserver.Connector); ok { ... }
type Connector interface {
	plugin.Connector

	// SendToDevice 直接向指定设备发送原始字节。
	SendToDevice(deviceId string, data []byte) error
	// SendToAll 向当前连接下的所有设备广播原始字节，返回成功发送的设备数量。
	SendToAll(data []byte) (sent int, err error)
	// GetConnectedDevices 返回当前连接的设备 ID 列表，顺序不保证稳定。
	GetConnectedDevices() []string
	// IsDeviceConnected 判断指定设备是否已有 TCP 连接映射。
	IsDeviceConnected(deviceId string) bool
}
