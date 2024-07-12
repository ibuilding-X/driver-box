package test

import (
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/internal/library"
	"github.com/ibuilding-x/driver-box/internal/logger"
	"go.uber.org/zap"
	"testing"
)

func TestDeviceEncode(t *testing.T) {
	Init()
	e := library.Driver().LoadLibrary("test_2")
	if e != nil {
		t.Error(e)
		return
	}
	result := library.Driver().DeviceEncode("test_2", library.DeviceEncodeRequest{
		DeviceId: "switch_1",
		Mode:     plugin.WriteMode,
		Points: []plugin.PointData{
			{
				PointName: "aa",
				Value:     int64(6),
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
	Init()
	e := library.Driver().LoadLibrary("test_1")
	if e != nil {
		t.Error(e)
		return
	}
	result := library.Driver().DeviceDecode("test_1", library.DeviceDecodeRequest{
		DeviceId: "test_1",
		Points: []plugin.PointData{
			{
				PointName: "aa",
				Value:     123123,
			},
			{
				PointName: "bb",
				Value:     1000,
			},
			{
				PointName: "cc",
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
