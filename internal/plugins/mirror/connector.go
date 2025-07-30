package mirror

import (
	"errors"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/internal/core"
	"github.com/ibuilding-x/driver-box/internal/logger"
	"go.uber.org/zap"
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

// Encode 编码数据
func (c *connector) Encode(deviceId string, mode plugin.EncodeMode, values ...plugin.PointData) (res interface{}, err error) {
	//本次操作的点位可能涉及到多个通讯设备，要先进行归类
	group := make(map[string]EncodeModel)
	for _, point := range values {
		//匹配镜像设备
		mirrorDevice, ok := c.mirrors[deviceId]
		if !ok {
			return nil, errors.New("mirror device not found")
		}
		//匹配原始设备
		rawDevice, ok := mirrorDevice[point.PointName]
		if !ok {
			return nil, errors.New("mirror pointName not found")
		}
		var points []plugin.PointData
		if _, ok := group[rawDevice.deviceId]; ok {
			points = group[rawDevice.deviceId].points
		} else {
			points = make([]plugin.PointData, 0)
		}
		points = append(points, plugin.PointData{
			PointName: rawDevice.pointName,
			Value:     point.Value,
		})
		group[rawDevice.deviceId] = EncodeModel{
			deviceId: rawDevice.deviceId,
			points:   points,
			mode:     mode,
		}
	}
	//group转数组
	models := make([]EncodeModel, 0)
	for _, model := range group {
		models = append(models, model)
	}
	return models, err
}

// Decode 数据来源于ExportTo
func (c *connector) Decode(raw interface{}) (res []plugin.DeviceData, err error) {
	//真实通讯设备的数据
	rawDeviceData := raw.(plugin.DeviceData)
	//镜像设备数据不支持二次镜像，否则配置失误时存在死循环风险
	if _, ok := c.mirrors[rawDeviceData.ID]; ok {
		return []plugin.DeviceData{}, err
	}
	pointMapping, ok := c.rawMapping[rawDeviceData.ID]
	//当前通讯设备不存在镜像设备
	if !ok {
		return []plugin.DeviceData{}, err
	}
	//镜像设备分组
	group := make(map[string]plugin.DeviceData)
	for _, point := range rawDeviceData.Values {
		mirrors, ok := pointMapping[point.PointName]
		if !ok {
			continue
		}
		for _, mirror := range mirrors {
			//镜像设备分组以存在，填充点位
			mirrorData, ok := group[mirror.ID]
			if !ok {
				mirrorData = plugin.DeviceData{
					ID:     mirror.ID,
					Values: make([]plugin.PointData, 0),
				}
			}
			//通讯设备对应同一镜像设备的多个点
			for _, pointData := range mirror.Values {
				mirrorData.Values = append(mirrorData.Values, plugin.PointData{
					PointName: pointData.PointName,
					Value:     point.Value,
				})
			}
			group[mirror.ID] = mirrorData
		}
	}
	for _, data := range group {
		res = append(res, data)
	}
	logger.Logger.Debug("mirror decode result", zap.Any("raw", rawDeviceData), zap.Any("mirror", res))
	return res, err
}
