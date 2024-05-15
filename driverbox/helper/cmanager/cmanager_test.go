package cmanager

import (
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"testing"
)

func TestManager(t *testing.T) {
	m := New()
	m.SetConfigPath("./drivers")
	m.SetConfigFileName("config.json")

	if err := m.LoadConfig(); err != nil {
		t.Error(err)
	}

	// 添加连接测试
	conn := map[string]any{
		"index":    0,
		"discover": true,
	}
	if err := m.AddConnection("modbus", "/dev/ttyUSB0", conn); err != nil {
		t.Error(err)
	}
	if err := m.AddConnection("bacnet", "/dev/ttyUSB1", conn); err != nil {
		t.Error(err)
	}

	// 添加模型
	model := config.DeviceModel{
		ModelBase: config.ModelBase{
			Name:    "7jhk541gjh57k14j517jk41",
			ModelID: "7jhk541gjh57k14j517jk41",
		},
		DevicePoints: []config.PointMap{
			{
				"name": "onOff",
			},
		},
		Devices: nil,
	}
	if err := m.AddModel("modbus", model); err != nil {
		t.Error(err)
	}

	// 添加设备
	device := config.Device{
		ID:            "device1",
		ModelName:     "7jhk541gjh57k14j517jk41",
		Description:   "device1",
		Ttl:           "15m",
		Tags:          nil,
		ConnectionKey: "/dev/ttyUSB0",
		Properties:    nil,
		DriverKey:     "",
	}
	if err := m.AddOrUpdateDevice(device); err != nil {
		t.Error(err)
	}
}
