package mirror

import (
	"errors"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
)

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
		points = append(points, point)
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
			if mirrorData, ok := group[mirror.ID]; ok {
				mirrorData.Values = append(mirrorData.Values, point)
				continue
			}
			//创建镜像设备数据
			group[mirror.ID] = plugin.DeviceData{
				ID: mirror.ID,
				Values: []plugin.PointData{
					point,
				},
			}
		}
	}
	for _, data := range group {
		res = append(res, data)
	}
	return res, err
}
