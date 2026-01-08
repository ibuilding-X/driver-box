package ui

import (
	"github.com/ibuilding-x/driver-box/pkg/driverbox/config"
	"github.com/ibuilding-x/driver-box/pkg/driverbox/helper"
	"github.com/ibuilding-x/driver-box/pkg/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/pkg/driverbox/restful"
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
	restful.HttpRouter.GET("/ui/", devices)
	restful.HttpRouter.GET("/ui/device/:deviceId", deviceDetail)
	//静态资源文件
	restful.HttpRouter.ServeFiles("/ui/css/*filepath", http.Dir("./res/ui/css"))
	restful.HttpRouter.ServeFiles("/ui/js/*filepath", http.Dir("./res/ui/js"))
	export.ready = true
	return nil
}
func NewExport() *Export {
	once.Do(func() {
		driverInstance = &Export{}
	})

	return driverInstance
}
func (export *Export) Destroy() error {
	export.ready = false
	return nil
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
