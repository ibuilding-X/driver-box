package linkage

import (
	"testing"
	"time"
)

var defaultConfig = Config{
	ID:              "ce96ca8e24955c9d16b4e3a8f571f9fa",
	Enable:          true,
	Name:            "default",
	Tags:            []string{"default"},
	Description:     "default linkage description",
	SilentPeriod:    0,
	Triggers:        []Trigger{},
	Conditions:      []Condition{},
	Actions:         []Action{},
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
