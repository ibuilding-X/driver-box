package mbserver

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
)

func TestStorage(t *testing.T) {
	storage := NewStorage()
	storage.EnableSort = true

	models := []Model{
		{
			Id:   "electricityMeter",
			Name: "电表",
			Properties: []Property{
				{
					Name:        "on",
					Description: "开关",
					ValueType:   ValueTypeBool,
					Access:      AccessReadWrite,
				},
				{
					Name:        "electricity",
					Description: "电量",
					ValueType:   ValueTypeFloat64,
					Access:      AccessRead,
				},
			},
		},
		{
			Id:   "vrf",
			Name: "多联机",
			Properties: []Property{
				{
					Name:        "on",
					Description: "开关",
					ValueType:   ValueTypeBool,
					Access:      AccessRead,
				},
				{
					Name:        "mode",
					Description: "运行模式",
					ValueType:   ValueTypeUint8,
					Access:      AccessReadWrite,
				},
				{
					Name:        "fanSpeed",
					Description: "风速",
					ValueType:   ValueTypeUint8,
					Access:      AccessReadWrite,
				},
				{
					Name:        "temperature",
					Description: "温度",
					ValueType:   ValueTypeFloat32,
					Access:      AccessRead,
				},
			},
		},
	}

	devices := []Device{
		{
			ModelId: "electricityMeter",
			Id:      "meter1",
		},
		{
			ModelId: "vrf",
			Id:      "vrf1",
		},
	}

	t.Run("Initialize", func(t *testing.T) {
		err := storage.Initialize(models, devices)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("SetProperty", func(t *testing.T) {
		if err := storage.SetProperty("meter1", "on", true); err != nil {
			t.Fatal(err)
		}
		if err := storage.SetProperty("meter1", "electricity", 9999.99); err != nil {
			t.Fatal(err)
		}

		if err := storage.SetProperty("vrf1", "on", true); err != nil {
			t.Fatal(err)
		}
		if err := storage.SetProperty("vrf1", "mode", 1); err != nil {
			t.Fatal(err)
		}
		if err := storage.SetProperty("vrf1", "fanSpeed", 3); err != nil {
			t.Fatal(err)
		}
		if err := storage.SetProperty("vrf1", "temperature", 26.5); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("GetProperty", func(t *testing.T) {
		if value, err := storage.GetProperty("meter1", "on"); err != nil {
			t.Fatal(err)
		} else if value != true {
			t.Fatal("on property should be true", value)
		}

		if value, err := storage.GetProperty("meter1", "electricity"); err != nil {
			t.Fatal(err)
		} else if value != 9999.99 {
			t.Fatal("electricity property should be 9999.99")
		}

		if value, err := storage.GetProperty("vrf1", "on"); err != nil {
			t.Fatal(err)
		} else if value != true {
			t.Fatal("on property should be true")
		}

		if value, err := storage.GetProperty("vrf1", "mode"); err != nil {
			t.Fatal(err)
		} else if value != uint8(1) {
			t.Fatal("mode property should be 1", reflect.ValueOf(value).Kind())
		}

		if value, err := storage.GetProperty("vrf1", "fanSpeed"); err != nil {
			t.Fatal(err)
		} else if value != uint8(3) {
			t.Fatal("fanSpeed property should be 3", value)
		}

		if value, err := storage.GetProperty("vrf1", "temperature"); err != nil {
			t.Fatal(err)
		} else if value != float32(26.5) {
			t.Fatal("temperature property should be 26.5", value)
		}
	})

	t.Run("DeviceMap", func(t *testing.T) {
		m := storage.DeviceMap()
		bs, _ := json.MarshalIndent(m, "", "\t")
		fmt.Println(string(bs))
	})
}
