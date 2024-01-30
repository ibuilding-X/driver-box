// 缓存助手

package helper

import (
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"go.uber.org/zap"
	"sync"
)

type DeviceProperties map[string]string

// CoreCache 核心缓存
var CoreCache coreCache

type cache struct {
	models  *sync.Map // name => config.Model
	devices *sync.Map // deviceSn => config.Device
	//设备属性
	deviceProperties   *sync.Map // deviceSn => map[string]map[string]string
	points             *sync.Map // deviceSn_pointName => config.Point
	devicePointPlugins *sync.Map // deviceSn_pointName => plugin key
	runningPlugins     *sync.Map // key => plugin.Plugin
	devicePointConn    *sync.Map // deviceSn_pointName => config.Device
	//根据tag分组存储的设备列表
	tagDevices *sync.Map // tag => []config.Device
}

func (c *cache) AddTag(tag string) (e error) {
	//TODO implement me
	panic("implement me")
}

// InitCoreCache 初始化核心缓存
func InitCoreCache(configMap map[string]config.Config) (err error) {
	c := &cache{
		models:             &sync.Map{},
		devices:            &sync.Map{},
		deviceProperties:   &sync.Map{},
		points:             &sync.Map{},
		devicePointPlugins: &sync.Map{},
		runningPlugins:     &sync.Map{},
		devicePointConn:    &sync.Map{},
	}
	CoreCache = c

	for key, _ := range configMap {
		for _, deviceModel := range configMap[key].DeviceModels {
			var model config.Model
			if raw, ok := c.models.Load(deviceModel.Name); ok {
				model = raw.(config.Model)
				if model.ModelBase.Name != deviceModel.ModelBase.Name ||
					model.ModelBase.ModelID != deviceModel.ModelBase.ModelID {
					return fmt.Errorf("conflict model base information: %v  %v",
						deviceModel.ModelBase, model.ModelBase)
				}
			} else {
				model = config.Model{
					ModelBase: deviceModel.ModelBase,
					Points:    map[string]config.Point{},
					Devices:   map[string]config.Device{},
				}
			}
			pointMap := make(map[string]config.Point)
			for _, devicePoint := range deviceModel.DevicePoints {
				point := devicePoint.ToPoint()
				pointMap[point.Name] = point
				if _, ok := model.Points[point.Name]; !ok {
					model.Points[point.Name] = point
				}
			}
			for _, device := range deviceModel.Devices {
				if device.DeviceSn == "" {
					Logger.Error("config error , device sn is empty", zap.Any("device", device))
					continue
				}
				deviceSn := device.DeviceSn
				device.ModelName = deviceModel.Name
				if _, ok := model.Devices[deviceSn]; !ok {
					model.Devices[deviceSn] = device
				}
				if deviceRaw, ok := c.devices.Load(deviceSn); !ok {
					c.devices.Store(deviceSn, device)
				} else {
					storedDeviceBase := deviceRaw.(config.Device)
					if storedDeviceBase.ModelName != device.ModelName {
						return fmt.Errorf("conflict model for device [%s]: %s -> %s", device.DeviceSn,
							device.ModelName, storedDeviceBase.ModelName)
					}
				}
				var properties map[string]DeviceProperties
				if raw, ok := c.deviceProperties.Load(deviceSn); ok {
					properties = raw.(map[string]DeviceProperties)
				} else {
					properties = make(map[string]DeviceProperties)
				}
				properties[configMap[key].ProtocolName+"_"+device.ConnectionKey] = device.Properties
				c.deviceProperties.Store(deviceSn, properties)
				for _, point := range pointMap {
					devicePointKey := deviceSn + "_" + point.Name
					if _, ok := c.points.Load(devicePointKey); ok {
						return fmt.Errorf("device %s duplicate point %s found", deviceSn, point.Name)
					}
					c.points.Store(devicePointKey, point)
					c.devicePointPlugins.Store(devicePointKey, key)
					c.devicePointConn.Store(devicePointKey, device)
				}

				//根据tag分组存储设备列表
				for _, tag := range device.Tags {
					if raw, ok := c.tagDevices.Load(tag); ok {
						devices := raw.([]config.Device)
						devices = append(devices, device)
						c.tagDevices.Store(tag, devices)
					} else {
						c.tagDevices.Store(tag, []config.Device{device})
					}
				}
			}
			c.models.Store(model.Name, model)
		}
	}

	return nil
}

// coreCache 核心缓存
type coreCache interface {
	GetModel(modelName string) (model config.Model, ok bool)                                     // model info
	GetDevice(deviceSn string) (device config.Device, ok bool)                                   // device info
	GetDeviceByDeviceAndPoint(deviceSn string, pointName string) (device config.Device, ok bool) // connection config
	//查询指定标签的设备列表
	GetDevicesByTag(tag string) (devices []config.Device)
	AddTag(tag string) (e error)                                                                        //
	GetPointByModel(modelName string, pointName string) (point config.Point, ok bool)                   // search point by model
	GetPointByDevice(deviceSn string, pointName string) (point config.Point, ok bool)                   // search point by device
	GetRunningPluginByDeviceAndPoint(deviceSn string, pointName string) (plugin plugin.Plugin, ok bool) // search plugin by device and point
	GetRunningPluginByKey(key string) (plugin plugin.Plugin, ok bool)                                   // search plugin by directory name
	AddRunningPlugin(key string, plugin plugin.Plugin)                                                  // add running plugin
	Models() (models []config.Model)                                                                    // all model
	Devices() (devices []config.Device)
	GetProtocolsByDevice(deviceSn string) (map[string]DeviceProperties, bool) // device protocols
	GetAllRunningPluginKey() (keys []string)                                  // get running plugin keys
}

func (c *cache) GetModel(modelName string) (model config.Model, ok bool) {
	if raw, exist := c.models.Load(modelName); exist {
		m, _ := raw.(config.Model)
		return m, true
	}
	return config.Model{}, false
}

func (c *cache) GetDevice(deviceSn string) (device config.Device, ok bool) {
	if raw, exist := c.devices.Load(deviceSn); exist {
		device, _ = raw.(config.Device)
		return device, true
	}
	return config.Device{}, false
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

func (c *cache) GetPointByModel(modelName string, pointName string) (point config.Point, ok bool) {
	if raw, ok := c.models.Load(modelName); ok {
		model := raw.(config.Model)
		if pointBase, ok := model.Points[pointName]; ok {
			return pointBase, true
		}
		return config.Point{}, false
	}
	return config.Point{}, false
}

func (c *cache) GetDevicesByTag(tag string) (devices []config.Device) {
	if raw, ok := c.tagDevices.Load(tag); ok {
		devices, _ = raw.([]config.Device)
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

func (c *cache) Devices() (devices []config.Device) {
	c.devices.Range(func(key, value any) bool {
		device, _ := value.(config.Device)
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
