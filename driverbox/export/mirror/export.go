package mirror

import (
	"github.com/ibuilding-x/driver-box/driverbox"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/event"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/driverbox/plugin/callback"
	"github.com/ibuilding-x/driver-box/internal/plugins/mirror"
	"go.uber.org/zap"
	"sync"
)

var driverInstance *Export
var once = &sync.Once{}

const MirrorPluginName = "mirror"

type Export struct {
	ready  bool
	plugin *mirror.Plugin
}

func (export *Export) Init() error {
	//注册镜像插件
	export.plugin = new(mirror.Plugin)
	driverbox.RegisterPlugin(MirrorPluginName, export.plugin)
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
	if export.plugin.VirtualConnector != nil {
		callback.OnReceiveHandler(export.plugin.VirtualConnector, deviceData)
	}
}

// 继承Export OnEvent接口
func (export *Export) OnEvent(eventCode string, key string, eventValue interface{}) error {
	if eventCode != event.EventCodeAddDevice {
		return nil
	}
	//自动生成镜像设备
	device, _ := helper.CoreCache.GetDevice(key)
	rawModel, ok := helper.CoreCache.GetModel(device.ModelName)
	if !ok {
		return nil
	}
	c := rawModel.Attributes["mirror"]
	if c == nil {
		return nil
	}
	mirrorConfig := new(autoMirrorConfig)
	if err := helper.Map2Struct(c, mirrorConfig); err != nil {
		return err
	}
	points := make([]config.PointMap, 0)
	for _, point := range mirrorConfig.Points {
		pointMap := config.PointMap{}
		for key, val := range point {
			pointMap[key] = val
		}
		pointMap["rawDevice"] = key
		points = append(points, pointMap)
	}
	modeName := rawModel.Name + "_mirror_" + key
	mirrorDevice := config.Device{
		ID:          "mirror_" + key,
		Description: device.Description,
		Ttl:         device.Ttl,
		Tags:        device.Tags,
		Properties:  device.Properties,
		DriverKey:   mirrorConfig.DriverKey,
		ModelName:   modeName,
	}
	mirrorModel := config.DeviceModel{
		ModelBase: config.ModelBase{
			Name:    modeName,
			ModelID: mirrorConfig.ModelId,
		},
		Devices: []config.Device{
			mirrorDevice,
		},
		DevicePoints: points,
	}
	e := helper.CoreCache.AddModel(MirrorPluginName, mirrorModel)
	helper.CoreCache.AddOrUpdateDevice(mirrorDevice)
	if e != nil {
		helper.Logger.Error("add mirror model error", zap.Error(e))
	}
	return nil
}

func (export *Export) IsReady() bool {
	return export.ready
}
