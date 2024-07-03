package websocket

import (
	"github.com/ibuilding-x/driver-box/driverbox/common"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
)

// Encode 编码数据，无需实现
func (a *connector) Encode(deviceSn string, mode plugin.EncodeMode, values ...plugin.PointData) (res interface{}, err error) {
	return nil, common.NotSupportEncode
}

// Decode 解码数据，调用动态脚本解析
func (a *connector) Decode(raw interface{}) (res []plugin.DeviceData, err error) {
	//return helper.CallLuaConverter(a.ls, "decode", raw)
	return nil, common.NotSupportDecode
}
