package mbslave

import (
	"fmt"
	"testing"
)

func TestDeviceHandler(t *testing.T) {
	var h DeviceHandler

	t.Run("NewDeviceHandler", func(t *testing.T) {
		h = NewDeviceHandler()
		if h == nil {
			t.Error("NewDeviceHandler should not return nil")
		}
	})

	t.Run("ImportModels", func(t *testing.T) {
		models := []Model{
			{
				ID: "20f35e630daf44dbfa4c3f68f5399d8c",
				Properties: []Property{
					{
						Name:      "on",
						ValueType: "uint16",
					},
					{
						Name:      "mode",
						ValueType: "uint16",
					},
					{
						Name:      "temperature",
						ValueType: "float32",
					},
					{
						Name:      "fanLevel",
						ValueType: "uint16",
					},
				},
			},
		}

		if err := h.ImportModels(models); err != nil {
			t.Error(err)
		}
	})

	t.Run("SetProperty", func(t *testing.T) {
		err := h.SetProperty(1, PropertyValue{
			Mid:      "20f35e630daf44dbfa4c3f68f5399d8c",
			Did:      "913f9c49dcb544e2087cee284f4a00b7",
			Property: "on",
			Value:    1,
		})
		if err != nil {
			t.Error(err)
		}

		err = h.SetProperty(1, PropertyValue{
			Mid:      "20f35e630daf44dbfa4c3f68f5399d8c",
			Did:      "913f9c49dcb544e2087cee284f4a00b7",
			Property: "mode",
			Value:    1,
		})
		if err != nil {
			t.Error(err)
		}

		err = h.SetProperty(1, PropertyValue{
			Mid:      "20f35e630daf44dbfa4c3f68f5399d8c",
			Did:      "913f9c49dcb544e2087cee284f4a00b7",
			Property: "temperature",
			Value:    27.5,
		})
		if err != nil {
			t.Error(err)
		}

		err = h.SetProperty(1, PropertyValue{
			Mid:      "20f35e630daf44dbfa4c3f68f5399d8c",
			Did:      "913f9c49dcb544e2087cee284f4a00b7",
			Property: "fanLevel",
			Value:    3,
		})
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("ReadHoldingRegisters", func(t *testing.T) {
		results, err := h.ReadHoldingRegisters(1, 0, 10)
		if err != nil {
			t.Error(err)
		}
		fmt.Println(results)
	})
}
