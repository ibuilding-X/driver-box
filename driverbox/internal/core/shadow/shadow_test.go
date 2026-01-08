package shadow

import (
	"testing"
	"time"
)

func TestShadow(t *testing.T) {
	shadow := NewDeviceShadow()

	t.Run("TestAddDevice", func(t *testing.T) {
		shadow.AddDevice("device", "model", 10*time.Second)
		shadow.AddDevice("device", "model2", 20*time.Second)
		shadow.AddDevice("device2", "model3", 30*time.Second)
	})

	t.Run("TestGetDevice", func(t *testing.T) {
		dev, ok := shadow.GetDevice("device")
		if !ok {
			t.Error("get device failed")
			return
		}
		if dev.ModelName != "model" && dev.TTL != "10s" {
			t.Error("get device failed, model or ttl is error")
			return
		}

		dev2, ok := shadow.GetDevice("device2")
		if !ok {
			t.Error("get device failed")
			return
		}
		if dev2.ModelName != "model3" && dev2.TTL != "30s" {
			t.Error("get device failed, model or ttl is error")
			return
		}
	})

	t.Run("TestHasDevice", func(t *testing.T) {
		if !shadow.HasDevice("device") {
			t.Error("has device failed")
			return
		}
		if !shadow.HasDevice("device2") {
			t.Error("has device failed")
			return
		}
	})

	t.Run("TestSetDevicePoint", func(t *testing.T) {
		for k, v := range testDataTypeMap {
			if err := shadow.SetDevicePoint("device", k, v); err != nil {
				t.Error("set device point failed", err)
				return
			}
		}
	})

	t.Run("TestGetDevicePoint", func(t *testing.T) {
		_ = shadow.SetOnline("device")
		for k, v := range testDataTypeMap {
			value, err := shadow.GetDevicePoint("device", k)
			if err != nil {
				t.Error("get device point failed", err)
				return
			}
			if value != v {
				t.Error("get device point failed, value is error")
				return
			}
		}
	})

	t.Run("TestGetDevicePoints", func(t *testing.T) {
		points, err := shadow.GetDevicePoints("device")
		if err != nil {
			t.Error("get device points failed", err)
			return
		}
		for k, _ := range testDataTypeMap {
			if testDataTypeMap[k] != points[k].Value {
				t.Error("get device points failed, value is error")
				return
			}
		}
	})

	t.Run("TestGetDeviceUpdateAt", func(t *testing.T) {
		updateAt, err := shadow.GetDeviceUpdateAt("device")
		if err != nil {
			t.Error("get device update at failed", err)
			return
		}
		if updateAt.IsZero() {
			t.Error("get device update at failed, update at is zero")
			return
		}
	})

	t.Run("TestGetDeviceStatus", func(t *testing.T) {
		_, err := shadow.GetDeviceStatus("device")
		if err != nil {
			t.Error("get device status failed", err)
			return
		}
	})

	t.Run("TestSetOnline", func(t *testing.T) {
		if err := shadow.SetOnline("device"); err != nil {
			t.Error("set online failed", err)
		}
	})

	t.Run("TestSetOffline", func(t *testing.T) {
		if err := shadow.SetOffline("device"); err != nil {
			t.Error("set offline failed", err)
		}
	})

	t.Run("TestMayBeOffline", func(t *testing.T) {
		if err := shadow.MayBeOffline("device"); err != nil {
			t.Error("may be offline failed", err)
		}
		if err := shadow.MayBeOffline("device"); err != nil {
			t.Error("may be offline failed", err)
		}
		if err := shadow.MayBeOffline("device"); err != nil {
			t.Error("may be offline failed", err)
		}
	})

	t.Run("TestSetOnlineChangeCallback", func(t *testing.T) {
		shadow.SetOnlineChangeCallback(func(device string, online bool) {
			t.Log("online change callback", device, online)
		})
	})

	t.Run("TestSetWritePointValue", func(t *testing.T) {
		for k, v := range testDataTypeMap {
			if err := shadow.SetWritePointValue("device", k, v); err != nil {
				t.Error("set write point value failed", err)
				return
			}
		}
	})

	t.Run("TestGetWritePointValue", func(t *testing.T) {
		for k, v := range testDataTypeMap {
			value, err := shadow.GetWritePointValue("device", k)
			if err != nil {
				t.Error("get write point value failed", err)
				return
			}
			if value != v {
				t.Error("get write point value failed, value is error")
				return
			}
		}
	})
}
