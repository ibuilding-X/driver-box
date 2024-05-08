package core

import (
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
func deviceDriverProcess(deviceId string, mode plugin.EncodeMode, pointData ...plugin.PointData) (*library.DeviceEncodeResult, bool) {
	device, ok := helper.CoreCache.GetDevice(deviceId)
	if !ok {
		helper.Logger.Error("unknown device", zap.Any("deviceId", device))
		return nil, false
	}
	if len(device.DriverKey) == 0 {
		return nil, false
	}
	return library.DeviceEncode(device.DriverKey, library.DeviceEncodeRequest{
		DeviceId: deviceId,
		Mode:     mode,
		Points:   pointData,
	}), true
}
