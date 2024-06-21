package shadow

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

var testDataTypeMap = map[string]any{
	"bool1":   true,
	"bool2":   false,
	"uint8":   10,
	"uint16":  10,
	"uint32":  10,
	"uint64":  10,
	"int8":    10,
	"int16":   10,
	"int32":   10,
	"int64":   10,
	"float32": 10.10,
	"float64": 10.10,
	"string":  "string",
}

func TestDevice(t *testing.T) {
	dev := newDevice("device", "model", 10*time.Second)

	t.Run("TestSetOffline", func(t *testing.T) {
		dev.setOnline(false)

		if dev.getOnline() != false {
			t.Error("device is not offline")
		}
	})

	t.Run("TestSetOnline", func(t *testing.T) {
		dev.setOnline(true)

		if dev.getOnline() != true {
			t.Error("device is not online")
		}
	})

	t.Run("TestMaybeOffline", func(t *testing.T) {
		dev.setOnline(true)
		dev.maybeOffline()
		dev.maybeOffline()
		dev.maybeOffline()
		if dev.getOnline() != false {
			t.Error("device is not offline")
		}
	})

	t.Run("TestSetPointValue", func(t *testing.T) {
		for k, v := range testDataTypeMap {
			dev.setPointValue(k, v)
		}
	})

	t.Run("TestGetPointValue", func(t *testing.T) {
		dev.setOnline(true)
		for k, v := range testDataTypeMap {
			value, ok := dev.getPointValue(k)
			if !ok {
				t.Error("failed to get point value")
			}
			if value != v {
				t.Error("point value is not equal")
			}
		}
	})

	t.Run("TestSetWritePointValue", func(t *testing.T) {
		for k, v := range testDataTypeMap {
			dev.setWritePointValue(k, v)
		}
	})

	t.Run("TestGetWritePointValue", func(t *testing.T) {
		for k, v := range testDataTypeMap {
			value, ok := dev.getWritePointValue(k)
			if !ok {
				t.Error("failed to get write point value")
			}
			if value != v {
				t.Error("point value is not equal")
			}
		}
	})

	t.Run("TestGetPoint", func(t *testing.T) {
		for k, v := range testDataTypeMap {
			value, ok := dev.getPoint(k)
			if !ok {
				t.Error("failed to get point")
			}
			if value.Value != v {
				t.Error("point value is not equal")
			}
		}
	})

	t.Run("TestToPublic", func(t *testing.T) {
		public := dev.toPublic()
		if public.ID != "device" {
			t.Error("device name is not equal")
		}
		if public.ModelName != "model" {
			t.Error("device model is not equal")
		}
		bs, _ := json.MarshalIndent(public, "", "\t")
		fmt.Println("-------------------- Public Device --------------------")
		fmt.Println(string(bs))
		fmt.Println("-------------------------------------------------------")
	})
}
