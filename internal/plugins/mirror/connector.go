package mirror

import (
	"errors"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/internal/core"
)

type connector struct {
	plugin *Plugin
	//镜像设备与真实设备的映射关系，镜像设备ID：{镜像点位:原始设备点位}
	//镜像设备的点位只会指向唯一的原始设备点位
	mirrors map[string]map[string]Device
	//真实设备点位与镜像设备的映射关系, rawDeviceId:{rawPointName:{mirrorDevice:[mirrorPoint]}}
	//原始设备点位可能指向多个镜像设备和多个点位
	//------------原始设备ID   原始点位    镜像点位------------
	rawMapping map[string]map[string][]plugin.DeviceData
}

// Release 虚拟链接，无需释放
func (c *connector) Release() (err error) {
	return
}

// ProtocolAdapter 协议适配器
func (p *connector) ProtocolAdapter() plugin.ProtocolAdapter {
	return p
}

// Send 发送请求
func (c *connector) Send(raw interface{}) (err error) {
	var e error
	models := raw.([]EncodeModel)
	for _, encodeModel := range models {
		switch encodeModel.mode {
		case plugin.WriteMode:
			e = core.SendBatchWrite(encodeModel.deviceId, encodeModel.points)
		case plugin.ReadMode:
			e = core.SendBatchRead(encodeModel.deviceId, encodeModel.points)
		default:
			return errors.New("unknown mode")
		}
		if e != nil {
			err = e
		}
	}
	return err

}
