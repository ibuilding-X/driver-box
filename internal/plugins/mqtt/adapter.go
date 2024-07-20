package mqtt

import (
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/internal/library"
)

func (conn *connector) Encode(deviceSn string, mode plugin.EncodeMode, values ...plugin.PointData) (res interface{}, err error) {
	return library.Protocol().Encode(conn.config.ProtocolKey, library.ProtocolEncodeRequest{
		DeviceId: deviceSn,
		Mode:     mode,
		Points:   values,
	})
}

// Decode 解析数据
func (conn *connector) Decode(raw interface{}) (res []plugin.DeviceData, err error) {
	return library.Protocol().Decode(conn.config.ProtocolKey, raw)
}
