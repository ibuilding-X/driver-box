package mirror

import (
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/event"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/driverbox/plugin/callback"
	"github.com/ibuilding-x/driver-box/internal/library"
	"github.com/ibuilding-x/driver-box/internal/plugins/mirror"
	"go.uber.org/zap"
	"os"
	"sync"
)

var driverInstance *Export
var once = &sync.Once{}

// 在 model attributes 中可以的key值
const MirrorTemplateName = "mirror_tpl"

const MirrorTemplateKeyName = "mirror_tpl_key"

type Export struct {
	ready  bool
	plugin *mirror.Plugin
}

func (export *Export) Init() error {
	if os.Getenv(config.ENV_EXPORT_MIRROR_ENABLED) == "false" {
		helper.Logger.Warn("mirror export is disabled")
		return nil
	}
	//注册镜像插件
	export.plugin = mirror.NewPlugin()

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
		//镜像设备仅存在一个虚拟连接
		virtualConnector, _ := export.plugin.Connector("", "")
		if virtualConnector != nil {
			callback.OnReceiveHandler(virtualConnector, deviceData)
		}
	}
	return nil
}

func (export *Export) autoCreateMirrorDevice(deviceId string) error {
	helper.Logger.Info("auto create mirror device checking", zap.String("deviceId", deviceId))
	//第一步：参数合法性校验
	device, ok := helper.CoreCache.GetDevice(deviceId)
	if !ok {
		helper.Logger.Info("auto create mirror device failed, device not found", zap.String("deviceId", deviceId))
		return nil
	}
	rawModel, ok := helper.CoreCache.GetModel(device.ModelName)
	if !ok {
		helper.Logger.Info("auto create mirror device failed, model not found", zap.String("deviceId", deviceId), zap.String("modelName", device.ModelName))
		return nil
	}
	c, err := export.getMirrorConfig(rawModel)
	if err != nil {
		helper.Logger.Error("auto create mirror device failed", zap.String("deviceId", deviceId), zap.Error(err))
		return err
	}
	if c == nil {
		helper.Logger.Info("auto create mirror device failed, no mirror config", zap.String("deviceId", deviceId), zap.Any("modeName", rawModel.Name))
		return nil
	}

	//第二步：生成设备、模型的内存结构
	mirrorConfig := new(autoMirrorConfig)
	if err := helper.Map2Struct(c, mirrorConfig); err != nil {
		return err
	}
	helper.Logger.Info("auto create mirror device", zap.String("deviceId", deviceId), zap.Any("mirrorConfig", mirrorConfig))
	modeName := rawModel.Name + "_mirror_" + deviceId
	properties := make(map[string]string)
	if device.Properties != nil {
		for key, val := range device.Properties {
			properties[key] = val
		}
	}
	properties[PropertyKeyAutoMirrorFrom] = deviceId
	delete(properties, PropertyKeyAutoMirrorTo)
	mirrorDevice := config.Device{
		ID:          "mirror_" + deviceId,
		Description: device.Description,
		Ttl:         device.Ttl,
		Tags:        device.Tags,
		Properties:  properties,
		DriverKey:   mirrorConfig.DriverKey,
		ModelName:   modeName,
	}

	helper.CoreCache.UpdateDeviceProperty(deviceId, PropertyKeyAutoMirrorTo, mirrorDevice.ID)
	if _, ok := helper.CoreCache.GetDevice(mirrorDevice.ID); ok {
		helper.Logger.Info("auto create mirror device ignore, device already exists", zap.String("deviceId", mirrorDevice.ID))
		return nil
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

	mirrorModel := config.DeviceModel{
		ModelBase: config.ModelBase{
			Name:        modeName,
			ModelID:     mirrorConfig.ModelId,
			Description: mirrorConfig.Description,
		},
		Devices: []config.Device{
			mirrorDevice,
		},
		DevicePoints: points,
	}
	//第三步：配置持久化
	e := helper.CoreCache.AddModel(mirror.ProtocolName, mirrorModel)
	if e != nil {
		helper.Logger.Error("add mirror model error", zap.Error(e))
		return e
	}
	e = helper.CoreCache.AddOrUpdateDevice(mirrorDevice)
	if e != nil {
		helper.Logger.Error("add mirror model error", zap.Error(e))
		return e
	}
	e = export.plugin.UpdateMirrorMapping(mirrorModel)
	if e == nil {
		helper.Logger.Info("auto create mirror device success", zap.String("deviceId", deviceId))
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
