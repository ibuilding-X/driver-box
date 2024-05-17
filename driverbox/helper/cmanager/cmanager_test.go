package cmanager

import (
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"testing"
)

func TestAddConnection(t *testing.T) {
	conn := map[string]any{
		"address":  "/dev/ttyS5",
		"discover": true,
	}
	if err := AddConnection("modbus", "/dev/ttyS5", conn); err != nil {
		t.Error(err)
	}
}

func TestAddModel(t *testing.T) {
	model := config.DeviceModel{
		ModelBase: config.ModelBase{
			Name:        "test_model_001",
			ModelID:     "test_model_001",
			Description: "测试模型",
		},
		DevicePoints: []config.PointMap{
			{
				"name": "onOff",
				"type": "int",
			},
		},
		Devices: nil,
	}
	if err := AddModel("modbus", model); err != nil {
		t.Error(err)
	}
}

func TestAddDevice(t *testing.T) {
	device := config.Device{
		ID:            "device_1",
		ModelName:     "test_model_001",
		Description:   "测试设备",
		Ttl:           "",
		Tags:          nil,
		ConnectionKey: "/dev/ttyS5",
		Properties:    nil,
		DriverKey:     "",
	}
	if err := AddOrUpdateDevice(device); err != nil {
		t.Error(err)
	}
}

func BenchmarkAddConnection(b *testing.B) {
	for i := 0; i < b.N; i++ {
		conn := map[string]any{
			"address":  "/dev/ttyS5",
			"discover": true,
		}
		if err := AddConnection("modbus", "/dev/ttyS5", conn); err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkAddModel(b *testing.B) {
	model := config.DeviceModel{
		ModelBase: config.ModelBase{
			Name:        "test_model_001",
			ModelID:     "test_model_001",
			Description: "测试模型",
		},
		DevicePoints: []config.PointMap{
			{
				"name": "onOff",
				"type": "int",
			},
		},
		Devices: nil,
	}
	for i := 0; i < b.N; i++ {
		if err := AddModel("modbus", model); err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkAddDevice(b *testing.B) {
	device := config.Device{
		ID:            "device_1",
		ModelName:     "test_model_001",
		Description:   "测试设备",
		Ttl:           "",
		Tags:          nil,
		ConnectionKey: "/dev/ttyS5",
		Properties:    nil,
		DriverKey:     "",
	}
	for i := 0; i < b.N; i++ {
		if err := AddOrUpdateDevice(device); err != nil {
			b.Error(err)
		}
	}
}
