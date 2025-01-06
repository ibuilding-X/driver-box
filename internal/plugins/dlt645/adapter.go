package dlt645

import (
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
)

// Decode 解码数据
func (c *connector) Decode(raw interface{}) (res []plugin.DeviceData, err error) {
	readValue, ok := raw.(plugin.PointReadValue)
	if !ok {
		return nil, fmt.Errorf("unexpected raw: %v", raw)
	}

	res = append(res, plugin.DeviceData{
		ID: readValue.ID,
		Values: []plugin.PointData{{
			PointName: readValue.PointName,
			Value:     readValue.Value,
		}},
	})
	return
}

// Encode 编码数据
func (c *connector) Encode(deviceId string, mode plugin.EncodeMode, values ...plugin.PointData) (res interface{}, err error) {
	if mode == plugin.WriteMode {
		return nil, err
	}

	device, ok := helper.CoreCache.GetDevice(deviceId)
	if !ok {
		return nil, fmt.Errorf("device [%s] not found", deviceId)
	}
	unitId, e := getMeterAddress(device.Properties)
	if e != nil {
		return nil, e
	}
	slave := c.devices[unitId]
	if slave == nil {
		return nil, fmt.Errorf("device [%s] not found", deviceId)
	}

	indexes := make(map[int]*pointGroup)
	var pointGroups []*pointGroup
	//寻找待读点位关联的pointGroup
	for _, readPoint := range values {
		ok = false
		for _, group := range slave.pointGroup {
			for _, point := range group.Points {
				if point.Name == readPoint.PointName {
					if _, ok := indexes[group.index]; !ok {
						indexes[group.index] = group
						pointGroups = append(pointGroups, group)
					}
					ok = true
					break
				}
			}
			//匹配成功
			if ok {
				break
			}
		}
	}

	//找到待读点所属的group
	return command{
		Mode:  BatchReadMode,
		Value: pointGroups,
	}, nil
}
