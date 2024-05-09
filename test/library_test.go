package test

import (
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/internal/library"
	"github.com/ibuilding-x/driver-box/internal/logger"
	"go.uber.org/zap"
	"path"
	"testing"
)

func TestDeviceEncode(t *testing.T) {
	logger.InitLogger("", "debug")
	e := library.LoadLibrary(library.DeviceDriver, "sensor_KSM34M-7H")
	if e != nil {
		t.Error(e)
		return
	}
	result := library.DeviceEncode("sensor_KSM34M-7H", library.DeviceEncodeRequest{
		DeviceId: "sensor_KSM34M-7H",
		Mode:     plugin.ReadMode,
		Points: []plugin.PointData{
			{
				PointName: "test",
			},
		},
	})
	if result.Error != nil {
		t.Error(result.Error)
		return
	}
	logger.Logger.Info("result", zap.Any("result", result))
}

func TestDeviceDecode(t *testing.T) {
	library.BaseDir = path.Join("res", "library")
	logger.InitLogger("", "debug")
	e := library.LoadLibrary(library.DeviceDriver, "sensor_KSM34M-7H")
	if e != nil {
		t.Error(e)
		return
	}
	result := library.DeviceDecode("sensor_KSM34M-7H", library.DeviceDecodeRequest{
		DeviceId: "sensor_KSM34M-7H",
		Points: []plugin.PointData{
			{
				PointName: "hcho",
				Value:     123123,
			},
			{
				PointName: "tvoc",
				Value:     1000,
			},
			{
				PointName: "aa",
				Value:     324,
			},
		},
	})
	if result.Error != nil {
		t.Error(result.Error)
		return
	}
	logger.Logger.Info("result", zap.Any("result", result))
}
