package cmanager

import (
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"testing"
)

var connections = []string{"/dev/ttyS0", "/dev/ttyS1", "/dev/ttyS2"}

var models = map[string]string{
	"model_1": "模型1",
	"model_2": "模型2",
	"model_3": "模型3",
}

var devices = map[string][]string{
	"model_1": {"device_1", "device_2", "device_3"},
	"model_2": {"device_4", "device_5", "device_6"},
	"model_3": {"device_7", "device_8", "device_9"},
}

func TestAddConnection(t *testing.T) {
	if err := LoadConfig(); err != nil {
		t.Error(err)
	}

	for _, connection := range connections {
		if err := AddConnection("modbus", connection, map[string]any{
			"address":  connection,
			"discover": true,
		}); err != nil {
			t.Error(err)
		}
	}
}

func TestAddModel(t *testing.T) {
	if err := LoadConfig(); err != nil {
		t.Error(err)
	}

	for key, _ := range models {
		if err := AddModel("modbus", config.DeviceModel{
			ModelBase: config.ModelBase{
				Name:        key,
				ModelID:     key,
				Description: models[key],
			},
		}); err != nil {
			t.Error(err)
		}
	}
}

func TestAddDevice(t *testing.T) {
	if err := LoadConfig(); err != nil {
		t.Error(err)
	}

	for model, _ := range devices {
		for _, device := range devices[model] {
			if err := AddOrUpdateDevice(config.Device{
				ID:            device,
				ModelName:     model,
				Description:   device,
				ConnectionKey: connections[0],
			}); err != nil {
				t.Error(err)
			}
		}
	}
}

func TestAddConfig(t *testing.T) {
	if err := LoadConfig(); err != nil {
		t.Error(err)
	}

	var c config.Config
	c.ProtocolName = "modbus"

	// 添加模型及设备
	c.DeviceModels = make([]config.DeviceModel, 0)
	for model, _ := range models {
		deviceList := make([]config.Device, 0)
		for _, device := range devices[model] {
			deviceList = append(deviceList, config.Device{
				ID:            device,
				ModelName:     model,
				Description:   device,
				ConnectionKey: connections[0],
			})
		}
		c.DeviceModels = append(c.DeviceModels, config.DeviceModel{
			ModelBase: config.ModelBase{
				Name:        model,
				ModelID:     model,
				Description: models[model],
			},
			Devices: deviceList,
		})
	}

	// 添加连接
	c.Connections = make(map[string]interface{})
	for _, connection := range connections {
		c.Connections[connection] = map[string]any{
			"address":  connection,
			"discover": true,
		}
	}

	if err := AddConfig(c); err != nil {
		t.Error(err)
	}
}

func TestRemoveConnection(t *testing.T) {
	if err := LoadConfig(); err != nil {
		t.Error(err)
	}

	if err := RemoveConnection(connections[0]); err != nil {
		t.Error(err)
	}
}

func TestRemoveDevice(t *testing.T) {
	if err := LoadConfig(); err != nil {
		t.Error(err)
	}

	if err := RemoveDeviceByID("device_1"); err != nil {
		t.Error(err)
	}

	if err := RemoveDevice("model_1", "device_2"); err != nil {
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
		DevicePoints: []config.Point{
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
