package websocket

import (
	"errors"
	"github.com/gorilla/websocket"
	"github.com/ibuilding-x/driver-box/driverbox/common"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/internal/library"
)

// Encode 编码数据，无需实现
func (c *connector) Encode(deviceId string, mode plugin.EncodeMode, values ...plugin.PointData) (res interface{}, err error) {
	payload, err := library.Protocol().Encode(c.config.DriverKey, library.ProtocolEncodeRequest{
		DeviceId: deviceId,
		Mode:     mode,
		Points:   values,
	})
	if err != nil {
		return nil, err
	}
	conn, ok := c.deviceMappingConn.Load(deviceId)
	if !ok {
		return nil, errors.New("device is disconnected")
	}
	return encodeStruct{
		payload:    payload,
		connection: conn.(*websocket.Conn),
	}, nil
}

// Decode 解码数据，调用动态脚本解析
func (a *connector) Decode(raw interface{}) (res []plugin.DeviceData, err error) {
	return nil, common.NotSupportDecode
}
