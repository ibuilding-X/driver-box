// 缓存助手

package helper

import (
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"sync"
)

type DeviceProperties map[string]string

// CoreCache 核心缓存
var CoreCache coreCache

type cache struct {
	models  *sync.Map // name => config.Model
	devices *sync.Map // deviceSn => config.DeviceBase
	//设备属性
	deviceProperties   *sync.Map // deviceSn => map[string]map[string]string
	points             *sync.Map // deviceSn_pointName => config.Point
	devicePointPlugins *sync.Map // deviceSn_pointName => plugin key
	runningPlugins     *sync.Map // key => plugin.Plugin
	devicePointConn    *sync.Map // deviceSn_pointName => config.Device
	//根据tag分组存储的设备列表
	tagDevices *sync.Map // tag => []config.DeviceBase
}

func (c *cache) AddTag(tag string) (e error) {
	//TODO implement me
	panic("implement me")
}

// InitCoreCache 初始化核心缓存
func InitCoreCache(configMap map[string]config.Config) (err error) {
	CoreCache := &cache{
		models:             &sync.Map{},
		devices:            &sync.Map{},
		deviceProperties:   &sync.Map{},
		points:             &sync.Map{},
		devicePointPlugins: &sync.Map{},
		runningPlugins:     &sync.Map{},
		devicePointConn:    &sync.Map{},
	}

	for key, _ := range configMap {
		for _, deviceModel := range configMap[key].DeviceModels {
			var model config.Model
			if raw, ok := CoreCache.models.Load(deviceModel.Name); ok {
				model = raw.(config.Model)
				if model.ModelBase.Name != deviceModel.ModelBase.Name ||
					model.ModelBase.ModelID != deviceModel.ModelBase.ModelID {
					return fmt.Errorf("conflict model base information: %v  %v",
						deviceModel.ModelBase, model.ModelBase)
				}
			} else {
				model = config.Model{
					ModelBase: deviceModel.ModelBase,
					Points:    map[string]config.PointBase{},
					Devices:   map[string]config.DeviceBase{},
				}
			}
			pointMap := make(map[string]config.Point)
			for _, devicePoint := range deviceModel.DevicePoints {
				point := devicePoint.ToPoint()
				pointMap[point.Name] = point
				if _, ok := model.Points[point.Name]; !ok {
					model.Points[point.Name] = point.PointBase
				}
			}
			for _, device := range deviceModel.Devices {
				deviceSn := device.DeviceSn
				deviceBase := device.DeviceBase
				deviceBase.ModelName = deviceModel.Name
				if _, ok := model.Devices[deviceSn]; !ok {
					model.Devices[deviceSn] = deviceBase
				}
				if deviceRaw, ok := CoreCache.devices.Load(deviceSn); !ok {
					CoreCache.devices.Store(deviceSn, deviceBase)
				} else {
					storedDeviceBase := deviceRaw.(config.DeviceBase)
					if storedDeviceBase.ModelName != deviceBase.ModelName {
						return fmt.Errorf("conflict model for device [%s]: %s -> %s", deviceBase.DeviceSn,
							deviceBase.ModelName, storedDeviceBase.ModelName)
					}
				}
				var properties map[string]DeviceProperties
				if raw, ok := CoreCache.deviceProperties.Load(deviceSn); ok {
					properties = raw.(map[string]DeviceProperties)
				} else {
					properties = make(map[string]DeviceProperties)
				}
				properties[configMap[key].ProtocolName+"_"+device.ConnectionKey] = device.Properties
				CoreCache.deviceProperties.Store(deviceSn, properties)
				for _, point := range pointMap {
					devicePointKey := deviceSn + "_" + point.Name
					if _, ok := CoreCache.points.Load(devicePointKey); ok {
						return fmt.Errorf("device %s duplicate point %s found", deviceSn, point.Name)
					}
					CoreCache.points.Store(devicePointKey, point)
					CoreCache.devicePointPlugins.Store(devicePointKey, key)
					CoreCache.devicePointConn.Store(devicePointKey, device)
				}

				//根据tag分组存储设备列表
				for _, tag := range device.Tags {
					if raw, ok := CoreCache.tagDevices.Load(tag); ok {
						devices := raw.([]config.DeviceBase)
						devices = append(devices, deviceBase)
						CoreCache.tagDevices.Store(tag, devices)
					} else {
						CoreCache.tagDevices.Store(tag, []config.DeviceBase{deviceBase})
					}
				}
			}
			CoreCache.models.Store(model.Name, model)
		}
	}

	return nil
}

// coreCache 核心缓存
type coreCache interface {
	GetModel(modelName string) (model config.ModelBase, ok bool)                                 // model info
	GetDevice(deviceSn string) (device config.DeviceBase, ok bool)                               // device info
	GetDeviceByDeviceAndPoint(deviceSn string, pointName string) (device config.Device, ok bool) // connection config
	//查询指定标签的设备列表
	GetDevicesByTag(tag string) (devices []config.DeviceBase)
	AddTag(tag string) (e error)                                                                        //
	GetPointByModel(modelName string, pointName string) (point config.PointBase, ok bool)               // search point by model
	GetPointByDevice(deviceSn string, pointName string) (point config.Point, ok bool)                   // search point by device
	GetRunningPluginByDeviceAndPoint(deviceSn string, pointName string) (plugin plugin.Plugin, ok bool) // search plugin by device and point
	GetRunningPluginByKey(key string) (plugin plugin.Plugin, ok bool)                                   // search plugin by directory name
	AddRunningPlugin(key string, plugin plugin.Plugin)                                                  // add running plugin
	Models() (models []config.Model)                                                                    // all model
	Devices() (devices []config.DeviceBase)
	GetProtocolsByDevice(deviceSn string) (map[string]DeviceProperties, bool) // device protocols
	GetAllRunningPluginKey() (keys []string)                                  // get running plugin keys
}

func (c *cache) GetModel(modelName string) (model config.ModelBase, ok bool) {
	if raw, exist := c.models.Load(modelName); exist {
		m, _ := raw.(config.Model)
		model = m.ModelBase
		return model, true
	}
	return config.ModelBase{}, false
}

func (c *cache) GetDevice(deviceSn string) (device config.DeviceBase, ok bool) {
	if raw, exist := c.devices.Load(deviceSn); exist {
		device, _ = raw.(config.DeviceBase)
		return device, true
	}
	return config.DeviceBase{}, false
}

func (c *cache) GetDeviceByDeviceAndConn(deviceSn, connectionKey string) (device config.Device, ok bool) {
	if raw, ok := c.deviceProperties.Load(deviceSn + "_" + connectionKey); ok {
		device, _ := raw.(config.Device)
		return device, true
	}
	return config.Device{}, false
}

func (c *cache) GetDeviceByDeviceAndPoint(deviceSn, pointName string) (device config.Device, ok bool) {
	if raw, ok := c.devicePointConn.Load(deviceSn + "_" + pointName); ok {
		device, _ = raw.(config.Device)
		return device, true
	}
	return config.Device{}, false
}

func (c *cache) GetPointByModel(modelName string, pointName string) (point config.PointBase, ok bool) {
	if raw, ok := c.models.Load(modelName); ok {
		model := raw.(config.Model)
		if pointBase, ok := model.Points[pointName]; ok {
			return pointBase, true
		}
		return config.PointBase{}, false
	}
	return config.PointBase{}, false
}

func (c *cache) GetDevicesByTag(tag string) (devices []config.DeviceBase) {
	if raw, ok := c.tagDevices.Load(tag); ok {
		devices, _ = raw.([]config.DeviceBase)
		return devices
	}
	return
}

func (c *cache) GetPointByDevice(deviceSn string, pointName string) (point config.Point, ok bool) {
	if raw, ok := c.points.Load(deviceSn + "_" + pointName); ok {
		return raw.(config.Point), true
	}
	return config.Point{}, false
}

func (c *cache) GetRunningPluginByDeviceAndPoint(deviceSn, pointName string) (plugin plugin.Plugin, ok bool) {
	if key, ok := c.devicePointPlugins.Load(fmt.Sprintf("%s_%s", deviceSn, pointName)); ok {
		return c.GetRunningPluginByKey(key.(string))
	}
	return nil, false
}

func (c *cache) GetRunningPluginByKey(key string) (p plugin.Plugin, ok bool) {
	if raw, ok := c.runningPlugins.Load(key); ok {
		p, _ = raw.(plugin.Plugin)
		return p, true
	}
	return nil, false
}

func (c *cache) AddRunningPlugin(key string, plugin plugin.Plugin) {
	c.runningPlugins.Store(key, plugin)
}

func (c *cache) Models() (models []config.Model) {
	c.models.Range(func(key, value any) bool {
		model, _ := value.(config.Model)
		models = append(models, model)
		return true
	})
	return
}

func (c *cache) Devices() (devices []config.DeviceBase) {
	c.devices.Range(func(key, value any) bool {
		device, _ := value.(config.DeviceBase)
		devices = append(devices, device)
		return true
	})
	return
}

func (c *cache) GetProtocolsByDevice(deviceSn string) (map[string]DeviceProperties, bool) {
	if raw, ok := c.deviceProperties.Load(deviceSn); ok {
		protocols, _ := raw.(map[string]DeviceProperties)
		return protocols, true
	}
	return nil, false
}

func (c *cache) GetAllRunningPluginKey() (keys []string) {
	c.runningPlugins.Range(func(key, value any) bool {
		k, _ := key.(string)
		keys = append(keys, k)
		return true
	})
	return
}
