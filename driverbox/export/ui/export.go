package ui

import (
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"net/http"
	"os"
	"sync"
)

var driverInstance *Export
var once = &sync.Once{}

type Export struct {
	ready bool
}

func (export *Export) Init() error {
	if os.Getenv(config.ENV_EXPORT_UI_ENABLED) == "false" {
		helper.Logger.Warn("driver-box ui is disabled")
		return nil
	}
	http.HandleFunc("/ui/", devices)
	export.ready = true
	return nil
}
func NewExport() *Export {
	once.Do(func() {
		driverInstance = &Export{}
	})

	return driverInstance
}

// 点位变化触发场景联动
func (export *Export) ExportTo(deviceData plugin.DeviceData) {
}

// 继承Export OnEvent接口
func (export *Export) OnEvent(eventCode string, key string, eventValue interface{}) error {

	return nil
}

func (export *Export) IsReady() bool {
	return export.ready
}
