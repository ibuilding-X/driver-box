package linkage

import (
	"testing"
	"time"
)

var defaultConfig = Config{
	ID:           "386de0e478445af87507657674203bab",
	Enable:       true,
	Name:         "default",
	Tags:         []string{"default"},
	Description:  "default linkage description",
	SilentPeriod: 0,
	Triggers: []Trigger{
		{
			Type: TriggerTypeDevicePoint,
			DevicePointTrigger: DevicePointTrigger{
				DeviceID:    "test_device_1",
				DevicePoint: "onOff",
				Condition:   ConditionSymbolEq,
				Value:       "1",
			},
		},
	},
	Conditions: []Condition{},
	Actions: []Action{
		{
			Type:      ActionTypeDevicePoint,
			Condition: nil,
			Sleep:     "",
			DevicePointAction: DevicePointAction{
				DeviceID: "test_device_2",
				Points: []DevicePoint{
					{
						Point: "onOff",
						Value: "1",
					},
				},
			},
		},
	},
	LastExecuteTime: time.Time{},
}

func TestLinkage(t *testing.T) {
	var ser Linkage

	t.Run("new", func(t *testing.T) {
		// 配置项
		options := NewOptions()
		options.SetDeviceReader(func(deviceID string, point string) (interface{}, error) {
			// todo something
			return nil, nil
		})
		options.SetDeviceWriter(func(deviceID string, points []DevicePoint) (err error) {
			// todo something
			return nil
		})
		options.SetCallback(func(result ExecutionResult) {
			// todo something
		})

		// 实例化
		ser = New(options)
	})

	t.Run("add", func(t *testing.T) {
		if err := ser.Add(defaultConfig); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("update", func(t *testing.T) {
		defaultConfig.Name = "edit_name"
		if err := ser.Update(defaultConfig); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("delete", func(t *testing.T) {

	})

	t.Run("trigger", func(t *testing.T) {

	})

	t.Run("queryAll", func(t *testing.T) {

	})

	t.Run("queryByID", func(t *testing.T) {

	})

	t.Run("queryByTag", func(t *testing.T) {

	})
}
