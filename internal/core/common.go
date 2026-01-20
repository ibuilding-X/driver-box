package core

import (
	"errors"
	"fmt"

	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/internal/cache"
	"github.com/ibuilding-x/driver-box/pkg/config"
	"github.com/ibuilding-x/driver-box/pkg/convutil"
	"github.com/ibuilding-x/driver-box/pkg/library"
	"go.uber.org/zap"
)

// serialNo 网关序列号
var Metadata = config.Metadata{
	SerialNo: "driver-box",
	Model:    "driver-box",
	Vendor:   "iBUILDING",
}

// 校验model有效性
func checkMode(mode plugin.EncodeMode) bool {
	switch mode {
	case plugin.ReadMode, plugin.WriteMode:
		return true
	default:
		return false
	}
}

// 点位值加工：设备驱动
func deviceDriverProcess(deviceId string, mode plugin.EncodeMode, pointData ...plugin.PointData) ([]plugin.PointData, error) {
	device, ok := cache.Get().GetDevice(deviceId)
	if !ok {
		helper.Logger.Error("unknown device", zap.Any("deviceId", device))
		return nil, errors.New("unknown device")
	}
	scaleEnable := len(device.DriverKey) == 0

	if mode == plugin.WriteMode {
		for i, p := range pointData {
			point, ok := cache.Get().GetPointByDevice(deviceId, p.PointName)
			if !ok {
				return nil, fmt.Errorf("not found point, point name is %s", p.PointName)
			}
			value, err := convutil.PointValue(p.Value, point.ValueType())
			if err != nil {
				return nil, err
			}
			if scaleEnable && point.Scale() != 0 {
				value, err = divideStrings(value, point.Scale())
				if err != nil {
					return nil, err
				}
			}
			pointData[i].Value = value
		}
	}

	if scaleEnable {
		return pointData, nil
	}
	result := library.Driver().DeviceEncode(device.DriverKey, library.DeviceEncodeRequest{
		DeviceId: deviceId,
		Mode:     mode,
		Points:   pointData,
	})
	return result.Points, result.Error
}

func divideStrings(value interface{}, scale float64) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v / scale, nil
	case int64:
		return float64(v) / scale, nil
	default:
		return 0, fmt.Errorf("cannot divide %T with float64", value)
	}
}
