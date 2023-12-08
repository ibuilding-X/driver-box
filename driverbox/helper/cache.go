// 缓存助手

package helper

import (
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"sync"
)

type ProtocolProperties map[string]string

// CoreCache 核心缓存
var CoreCache coreCache

type cache struct {
	models             *sync.Map // name => config.Model
	devices            *sync.Map // deviceName => config.DeviceBase
	deviceProtocols    *sync.Map // deviceName => map[string]map[string]string
	points             *sync.Map // deviceName_pointName => config.Point
	devicePointPlugins *sync.Map // deviceName_pointName => plugin key
	runningPlugins     *sync.Map // key => plugin.Plugin
	devicePointConn    *sync.Map // deviceName_pointName => config.Device
}

// InitCoreCache 初始化核心缓存
func InitCoreCache(configMap map[string]config.Config) (err error) {
	c := &cache{
		models:             &sync.Map{},
		devices:            &sync.Map{},
		deviceProtocols:    &sync.Map{},
		points:             &sync.Map{},
		devicePointPlugins: &sync.Map{},
		runningPlugins:     &sync.Map{},
		devicePointConn:    &sync.Map{},
	}

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
				deviceName := device.Name
				deviceBase := device.DeviceBase
				deviceBase.ModelName = deviceModel.Name
				if _, ok := model.Devices[deviceName]; !ok {
					model.Devices[deviceName] = deviceBase
				}
				if deviceRaw, ok := c.devices.Load(deviceName); !ok {
					c.devices.Store(deviceName, deviceBase)
				} else {
					storedDeviceBase := deviceRaw.(config.DeviceBase)
					if storedDeviceBase.ModelName != deviceBase.ModelName {
						return fmt.Errorf("conflict model for device [%s]: %s -> %s", deviceBase.Name,
							deviceBase.ModelName, storedDeviceBase.ModelName)
					}
				}
				var protocols map[string]ProtocolProperties
				if raw, ok := c.deviceProtocols.Load(deviceName); ok {
					protocols = raw.(map[string]ProtocolProperties)
				} else {
					protocols = make(map[string]ProtocolProperties)
				}
				protocols[configMap[key].ProtocolName+"_"+device.ConnectionKey] = device.Protocol
				c.deviceProtocols.Store(deviceName, protocols)
				for _, point := range pointMap {
					devicePointKey := deviceName + "_" + point.Name
					if _, ok := c.points.Load(devicePointKey); ok {
						return fmt.Errorf("device %s duplicate point %s found", deviceName, point.Name)
					}
					c.points.Store(devicePointKey, point)
					c.devicePointPlugins.Store(devicePointKey, key)
					c.devicePointConn.Store(devicePointKey, device)
				}
			}
			c.models.Store(model.Name, model)
		}
	}

	CoreCache = c
	return nil
}

// coreCache 核心缓存
type coreCache interface {
	GetModel(modelName string) (model config.ModelBase, ok bool)                                          // model info
	GetDevice(deviceName string) (device config.DeviceBase, ok bool)                                      // device info
	GetDeviceByDeviceAndPoint(deviceName string, pointName string) (device config.Device, ok bool)        // connection config
	GetPointByModel(modelName string, pointName string) (point config.PointBase, ok bool)                 // search point by model
	GetPointByDevice(deviceName string, pointName string) (point config.Point, ok bool)                   // search point by device
	GetRunningPluginByDeviceAndPoint(deviceName string, pointName string) (plugin plugin.Plugin, ok bool) // search plugin by device and point
	GetRunningPluginByKey(key string) (plugin plugin.Plugin, ok bool)                                     // search plugin by directory name
	AddRunningPlugin(key string, plugin plugin.Plugin)                                                    // add running plugin
	Models() (models []config.Model)                                                                      // all model
	Devices() (devices []config.DeviceBase)
	GetProtocolsByDevice(deviceName string) (map[string]ProtocolProperties, bool) // device protocols
	GetAllRunningPluginKey() (keys []string)                                      // get running plugin keys
}

func (c *cache) GetModel(modelName string) (model config.ModelBase, ok bool) {
	if raw, exist := c.models.Load(modelName); exist {
		m, _ := raw.(config.Model)
		model = m.ModelBase
		return model, true
	}
	return config.ModelBase{}, false
}

func (c *cache) GetDevice(deviceName string) (device config.DeviceBase, ok bool) {
	if raw, exist := c.devices.Load(deviceName); exist {
		device, _ = raw.(config.DeviceBase)
		return device, true
	}
	return config.DeviceBase{}, false
}

func (c *cache) GetDeviceByDeviceAndConn(deviceName, connectionKey string) (device config.Device, ok bool) {
	if raw, ok := c.deviceProtocols.Load(deviceName + "_" + connectionKey); ok {
		device, _ := raw.(config.Device)
		return device, true
	}
	return config.Device{}, false
}

func (c *cache) GetDeviceByDeviceAndPoint(deviceName, pointName string) (device config.Device, ok bool) {
	if raw, ok := c.devicePointConn.Load(deviceName + "_" + pointName); ok {
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

func (c *cache) GetPointByDevice(deviceName string, pointName string) (point config.Point, ok bool) {
	if raw, ok := c.points.Load(deviceName + "_" + pointName); ok {
		return raw.(config.Point), true
	}
	return config.Point{}, false
}

func (c *cache) GetRunningPluginByDeviceAndPoint(deviceName, pointName string) (plugin plugin.Plugin, ok bool) {
	if key, ok := c.devicePointPlugins.Load(fmt.Sprintf("%s_%s", deviceName, pointName)); ok {
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

func (c *cache) GetProtocolsByDevice(deviceName string) (map[string]ProtocolProperties, bool) {
	if raw, ok := c.deviceProtocols.Load(deviceName); ok {
		protocols, _ := raw.(map[string]ProtocolProperties)
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
