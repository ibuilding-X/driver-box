package dlt645

import (
	"encoding/json"
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"sort"
)

// Decode 解码数据
func (c *connector) Decode(raw interface{}) (res []plugin.DeviceData, err error) {
	readValue, ok := raw.(plugin.PointReadValue)
	if !ok {
		return nil, fmt.Errorf("unexpected raw: %v", raw)
	}

	if c.ScriptEnable {
		resBytes, err := json.Marshal(readValue)
		if err != nil {
			return nil, fmt.Errorf("marshal result [%v] error: %v", res, err)
		}
		return helper.CallLuaConverter(c.Ls, "decode", string(resBytes))
	} else {
		res = append(res, plugin.DeviceData{
			ID: readValue.ID,
			Values: []plugin.PointData{{
				PointName: readValue.PointName,
				Value:     readValue.Value,
			}},
		})
	}
	return
}

// Encode 编码数据
func (c *connector) Encode(deviceId string, mode plugin.EncodeMode, values ...plugin.PointData) (res interface{}, err error) {
	if mode == plugin.WriteMode {
		writeValues, err := c.batchWriteEncode(deviceId, values)
		return command{
			Mode:  plugin.WriteMode,
			Value: writeValues,
		}, err
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

func (c *connector) batchWriteEncode(deviceId string, points []plugin.PointData) ([]*writeValue, error) {
	values := make([]*writeValue, 0)
	for _, p := range points {
		wv, err := c.getWriteValue(deviceId, p)
		if err != nil {
			return values, err
		}
		values = append(values, &wv)
	}
	//按照address排序
	sort.Slice(values, func(i, j int) bool {
		return values[i].Address < values[j].Address
	})

	mergedValues := make([]*writeValue, 0)
	var preValue *writeValue
	for _, v := range values {
		//仅保持寄存器支持批量
		if v.RegisterType != HoldingRegister {
			mergedValues = append(mergedValues, v)
			continue
		}
		if preValue == nil {
			preValue = v
			mergedValues = append(mergedValues, v)
			continue
		}

		//批量下发必须为连续地址
		if int(preValue.Address)+len(preValue.Value) != int(v.Address) {
			preValue = v
			mergedValues = append(mergedValues, v)
			continue
		}
		//超过批量写支持的字节长度范围
		batchLen := len(preValue.Value) + len(v.Value)
		if uint16(batchLen) > c.config.BatchWriteLen {
			preValue = v
			mergedValues = append(mergedValues, v)
			continue
		}
		//合并数据
		bytes := make([]uint16, batchLen)
		copy(bytes, preValue.Value)
		copy(bytes[len(preValue.Value):], v.Value)
		preValue.Value = bytes
	}
	return mergedValues, nil
}

func (c *connector) getWriteValue(deviceId string, pointData plugin.PointData) (writeValue, error) {
	return writeValue{}, nil
}
