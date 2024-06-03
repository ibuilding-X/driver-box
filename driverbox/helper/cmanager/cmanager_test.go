package cmanager

import (
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"testing"
)

func TestAddConnection(t *testing.T) {
	if err := LoadConfig(); err != nil {
		t.Error(err)
	}

	conn := map[string]any{
		"address":  "/dev/ttyS5",
		"discover": true,
	}
	if err := AddConnection("modbus", "/dev/ttyS5", conn); err != nil {
		t.Error(err)
	}
}

func TestAddModel(t *testing.T) {
	if err := LoadConfig(); err != nil {
		t.Error(err)
	}

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
	if err := LoadConfig(); err != nil {
		t.Error(err)
	}

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

func TestAddConfig(t *testing.T) {
	if err := LoadConfig(); err != nil {
		t.Error(err)
	}

	c := config.Config{
		DeviceModels: []config.DeviceModel{
			{
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
				Devices: []config.Device{
					{
						ID:            "device_2",
						ModelName:     "test_model_001",
						Description:   "测试设备2",
						Ttl:           "",
						Tags:          nil,
						ConnectionKey: "/dev/ttyS0",
						Properties:    nil,
						DriverKey:     "",
					},
				},
			},
			{
				ModelBase: config.ModelBase{
					Name:        "test_model_002",
					ModelID:     "test_model_002",
					Description: "测试模型2",
				},
				DevicePoints: []config.PointMap{
					{
						"name": "onOff",
						"type": "int",
					},
				},
				Devices: nil,
			},
		},
		Connections: map[string]any{
			"/dev/ttyS0": map[string]any{
				"address":  "/dev/ttyS0",
				"discover": true,
			},
		},
		ProtocolName: "modbus",
	}

	if err := AddConfig(c); err != nil {
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
