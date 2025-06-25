package mirror

import (
	"errors"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/event"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/helper/cmanager"
	"github.com/ibuilding-x/driver-box/driverbox/library"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/driverbox/plugin/callback"
	"github.com/ibuilding-x/driver-box/internal/logger"
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
		res, err := export.plugin.Decode(deviceData)
		if err != nil {
			return err
		}
		callback.ExportTo(res)
	case event.EventCodeDeviceStatus:
		// 设备状态变更事件
		mirrorDeviceID := "mirror_" + key
		if helper.DeviceShadow.HasDevice(mirrorDeviceID) {
			if online, ok := eventValue.(bool); ok {
				if online {
					_ = helper.DeviceShadow.SetOnline(mirrorDeviceID)
				} else {
					_ = helper.DeviceShadow.SetOffline(mirrorDeviceID)
				}
			}
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
		ModelName:   rawModel.Name + "_mirror_" + deviceId,
	}

	helper.CoreCache.UpdateDeviceProperty(deviceId, PropertyKeyAutoMirrorTo, mirrorDevice.ID)
	if _, ok := helper.CoreCache.GetDevice(mirrorDevice.ID); ok {
		helper.Logger.Info("auto create mirror device ignore, device already exists", zap.String("deviceId", mirrorDevice.ID))
		return nil
	}

	//加载模型库资源
	mirrorModel, err := library.Model().LoadLibrary(mirrorConfig.ModelKey)
	if err != nil {
		helper.Logger.Error("auto create mirror device failed, modelKey not exists", zap.String("deviceId", deviceId), zap.String("modelKey", mirrorConfig.ModelKey), zap.Error(err))
		return err
	}
	mirrorModel.Name = mirrorDevice.ModelName
	mirrorModel.Description = mirrorConfig.Description
	mirrorModel.Devices = []config.Device{mirrorDevice}

	points := make([]config.PointMap, 0)
	for _, point := range mirrorConfig.Points {
		pointName, ok := point["name"]
		if !ok || len(pointName) == 0 {
			helper.Logger.Error("auto create mirror device failed, point name is nil", zap.Any("mirrorConfig", mirrorConfig), zap.String("deviceId", deviceId))
			return nil
		}
		//根据镜像模版中定义的点名，找到镜像模型的点位配置
		for _, mirrorPoint := range mirrorModel.DevicePoints {
			if mirrorPoint["name"] != pointName {
				continue
			}
			//镜像模版中的点位配置作为高优先级，覆盖镜像模型的点位配置
			for k, v := range point {
				mirrorPoint[k] = v
			}
			mirrorPoint["rawDevice"] = deviceId
			points = append(points, mirrorPoint)
			break
		}
	}
	mirrorModel.DevicePoints = points

	//第三步：配置持久化
	e := helper.CoreCache.AddModel(mirror.ProtocolName, mirrorModel.ToModel())
	if e != nil {
		helper.Logger.Error("add mirror model error", zap.Error(e))
		return e
	}
	e = helper.CoreCache.AddOrUpdateDevice(mirrorDevice)
	if e != nil {
		helper.Logger.Error("add mirror model error", zap.Error(e))
		return e
	}
	//ready为false，说明不存在mirror目录
	if export.plugin.IsReady() {
		e = export.plugin.UpdateMirrorMapping(mirrorModel)
	} else {
		logger.Logger.Info("add mirror model success, but mirror plugin is not ready. will initialize...")
		c, ok := cmanager.GetConfig(mirror.ProtocolName)
		if !ok {
			logger.Logger.Info("mirror plugin initialize fail")
			return errors.New("mirror config not found")
		}
		export.plugin.Initialize(nil, c, nil)
		// 缓存插件
		helper.CoreCache.AddRunningPlugin(mirror.ProtocolName, export.plugin)
		logger.Logger.Info("mirror plugin initialize success")
	}

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
