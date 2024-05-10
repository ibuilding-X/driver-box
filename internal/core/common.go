package core

import (
	"errors"
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/internal/library"
	"go.uber.org/zap"
)

// 校验model有效性
func checkMode(mode plugin.EncodeMode) bool {
	switch mode {
	case plugin.ReadMode, plugin.WriteMode:
		return true
	default:
		return false
	}
}

// 点位值加工：精度配置化
func pointScaleProcess(pointData *plugin.PointData, point config.Point) error {
	if point.Scale == 0 {
		return nil
	}
	value, err := helper.ConvPointType(pointData.Value, point.ValueType)
	if err != nil {
		return err
	}
	if point.Scale != 0 {
		value, err = divideStrings(value, point.Scale)
		if err != nil {
			return err
		}
	}
	pointData.Value = value
	return nil
}

// 点位值加工：设备驱动
func deviceDriverProcess(deviceId string, mode plugin.EncodeMode, pointData ...plugin.PointData) ([]plugin.PointData, error) {
	device, ok := helper.CoreCache.GetDevice(deviceId)
	if !ok {
		helper.Logger.Error("unknown device", zap.Any("deviceId", device))
		return nil, errors.New("unknown device")
	}
	scaleEnable := len(device.DriverKey) == 0

	if mode == plugin.WriteMode {
		for _, p := range pointData {
			point, ok := helper.CoreCache.GetPointByDevice(deviceId, p.PointName)
			if !ok {
				return nil, fmt.Errorf("not found point, point name is %s", p.PointName)
			}
			value, err := helper.ConvPointType(p.Value, point.ValueType)
			if err != nil {
				return nil, err
			}
			if scaleEnable && point.Scale != 0 {
				value, err = divideStrings(value, point.Scale)
				if err != nil {
					return nil, err
				}
			}
			p.Value = value
		}
	}

	if scaleEnable {
		return pointData, nil
	}
	result := library.DeviceEncode(device.DriverKey, library.DeviceEncodeRequest{
		DeviceId: deviceId,
		Mode:     mode,
		Points:   pointData,
	})
	return result.Points, result.Error
}
