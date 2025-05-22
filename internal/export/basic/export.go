package basic

import (
	"fmt"
	"os"
	"sync"

	"github.com/ibuilding-x/driver-box/internal/core"

	"github.com/google/uuid"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
)

var driverInstance *Export
var once = &sync.Once{}

// 设备自动发现插件
type Export struct {
	ready bool
}

func (export *Export) Init() error {
	// 检查并生成唯一码文件
	const uniqueCodeFile = ".driverbox_serial_no"
	if _, err := os.Stat(uniqueCodeFile); err == nil {
		// 文件存在则读取内容
		content, err := os.ReadFile(uniqueCodeFile)
		if err != nil {
			return fmt.Errorf("failed to read unique code file: %v", err)
		}
		core.Metadata.SerialNo = string(content)
	} else if os.IsNotExist(err) {
		// 生成UUID作为唯一码
		uniqueCode := uuid.New().String()
		if err := os.WriteFile(uniqueCodeFile, []byte(uniqueCode), 0644); err != nil {
			return fmt.Errorf("failed to write unique code file: %v", err)
		}
		core.Metadata.SerialNo = uniqueCode
	}

	export.ready = true
	registerApi()
	go udpDiscover()
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
