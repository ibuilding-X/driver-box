package discover

import (
	"os"
	"sync"

	"github.com/ibuilding-x/driver-box/v2/driverbox"
	"github.com/ibuilding-x/driver-box/v2/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/v2/pkg/config"
	"github.com/ibuilding-x/driver-box/v2/pkg/convutil"
	"github.com/ibuilding-x/driver-box/v2/pkg/event"
	"github.com/ibuilding-x/driver-box/v2/pkg/library"
	"go.uber.org/zap"
)

var driverInstance *export
var once = &sync.Once{}

// 设备自动发现插件
type export struct {
	ready bool
}

func EnableExport() {
	driverbox.EnableExport(newExport())
}
func (export *export) Init() error {
	if os.Getenv(config.ENV_EXPORT_DISCOVER_ENABLED) == "false" {
		driverbox.Log().Warn("discover export is disabled")
		return nil
	}
	export.ready = true
	return nil
}
func newExport() *export {
	once.Do(func() {
		driverInstance = &export{}
	})
	return driverInstance
}

func (export *export) Destroy() error {
	export.ready = false
	return nil
}

// 点位变化触发场景联动
func (export *export) ExportTo(deviceData plugin.DeviceData) {
	//设备点位已通过
	//if export.plugin.VirtualConnector != nil && len(deviceData.Events) > 0 {
	//	driverbox.Log().Info("export to virtual connector", zap.Any("deviceData", deviceData))
	//	callback.OnReceiveHandler(export.plugin.VirtualConnector, deviceData)
	//}
}

// 继承Export OnEvent接口
func (export *export) OnEvent(eventCode event.EventCode, key string, eventValue interface{}) error {
	switch eventCode {
	case event.DeviceDiscover:
		return export.deviceAutoDiscover(key, eventValue)
	}
	return nil
}

// 设备自动发现
func (export *export) deviceAutoDiscover(deviceId string, value interface{}) error {
	driverbox.Log().Info("device auto discover", zap.Any("deviceId", deviceId), zap.Any("value", value))
	deviceDiscover := DeviceDiscover{}
	if err := convutil.Struct(value, &deviceDiscover); err != nil {
		driverbox.Log().Error("device auto discover conv2struct error", zap.String("deviceId", deviceId), zap.Any("value", value), zap.Any("error", err))
		return err
	}
	model, err := library.Model().LoadLibrary(deviceDiscover.ModelKey)
	if err != nil {
		driverbox.Log().Error("device auto discover load model error", zap.String("deviceId", deviceId), zap.Any("value", value), zap.Any("error", err))
		return err
	}
	//通过 modelKey 添加的统一模型 Name
	if len(deviceDiscover.ModelName) > 0 {
		model.Name = deviceDiscover.ModelName
	} else {
		model.Name = deviceDiscover.ProtocolName + "_" + deviceDiscover.ModelKey
	}

	//覆盖模型点位属性
	if len(deviceDiscover.Model) > 0 {
		points := make([]config.Point, 0)
		for _, point := range model.DevicePoints {
			pointName := point["name"].(string)
			luaPoint, ok := deviceDiscover.Model[pointName]
			if ok {
				for k, v := range luaPoint {
					point[k] = v
				}
			}
			points = append(points, point)
			delete(deviceDiscover.Model, pointName)
		}
		//取并集
		for pointName, prop := range deviceDiscover.Model {
			point := make(config.Point)
			point["name"] = pointName
			for k, v := range prop {
				point[k] = v
			}
			points = append(points, point)
		}
		model.DevicePoints = points
	}

	err = driverbox.CoreCache().AddModel(deviceDiscover.ProtocolName, model)
	if err != nil {
		driverbox.Log().Error("device auto discover add model error", zap.String("deviceId", deviceId), zap.Any("value", value), zap.Any("error", err))
		return err
	}
	//添加设备
	deviceDiscover.Device.ModelName = model.Name
	deviceDiscover.Device.ConnectionKey = deviceDiscover.ConnectionKey
	err = driverbox.CoreCache().AddOrUpdateDevice(deviceDiscover.Device)
	if err != nil {
		driverbox.Log().Error("device auto discover add device error", zap.String("deviceId", deviceId), zap.Any("value", value), zap.Any("error", err))
		return err
	}

	return nil
}

func (export *export) IsReady() bool {
	return export.ready
}
