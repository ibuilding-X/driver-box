package mirror

import (
	"github.com/ibuilding-x/driver-box/driverbox"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/event"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/driverbox/plugin/callback"
	"github.com/ibuilding-x/driver-box/internal/library"
	"github.com/ibuilding-x/driver-box/internal/plugins/mirror"
	"go.uber.org/zap"
	"sync"
)

var driverInstance *Export
var once = &sync.Once{}

const MirrorPluginName = "mirror"

// 在 model attributes 中可以的key值
const MirrorTemplateName = "mirror_tpl"

const MirrorTemplateKeyName = "mirror_tpl_key"

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
	//设备点位已通过
	//if export.plugin.VirtualConnector != nil && len(deviceData.Events) > 0 {
	//	helper.Logger.Info("export to virtual connector", zap.Any("deviceData", deviceData))
	//	callback.OnReceiveHandler(export.plugin.VirtualConnector, deviceData)
	//}
}

// 继承Export OnEvent接口
func (export *Export) OnEvent(eventCode string, key string, eventValue interface{}) error {
	switch eventCode {
	case event.EventCodeAddDevice:
		return export.autoCreateMirrorDevice(key)
	case event.EventCodeWillExportTo:
		deviceData := eventValue.(plugin.DeviceData)
		if export.plugin.VirtualConnector != nil {
			callback.OnReceiveHandler(export.plugin.VirtualConnector, deviceData)
		}
	}
	return nil
}

func (export *Export) autoCreateMirrorDevice(deviceId string) error {
	//自动生成镜像设备
	device, _ := helper.CoreCache.GetDevice(deviceId)
	rawModel, ok := helper.CoreCache.GetModel(device.ModelName)
	if !ok {
		return nil
	}
	c, err := export.getMirrorConfig(rawModel)
	if err != nil {
		return err
	}
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
		pointMap["rawDevice"] = deviceId
		points = append(points, pointMap)
	}
	modeName := rawModel.Name + "_mirror_" + deviceId
	mirrorDevice := config.Device{
		ID:          "mirror_" + deviceId,
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
	//若模型已存在，则忽略
	e := helper.CoreCache.AddModel(MirrorPluginName, mirrorModel)
	if e != nil {
		helper.Logger.Error("add mirror model error", zap.Error(e))
		return e
	}
	e = helper.CoreCache.AddOrUpdateDevice(mirrorDevice)
	if e != nil {
		helper.Logger.Error("add mirror model error", zap.Error(e))
	}
	return e
}

// 获取模型中关联的镜像配置
func (export *Export) getMirrorConfig(rawModel config.Model) (interface{}, error) {
	c := rawModel.Attributes[MirrorTemplateName]
	if c != nil {
		return c, nil
	}
	tmpKey := rawModel.Attributes[MirrorTemplateKeyName]
	if tmpKey == nil {
		return nil, nil
	}
	return library.Mirror().LoadLibrary(tmpKey.(string))
}

func (export *Export) IsReady() bool {
	return export.ready
}
