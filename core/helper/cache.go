// 缓存助手

package helper

import (
	"driver-box/core/config"
	"driver-box/core/contracts"
	"sync"
)

// CoreCache 核心缓存
var CoreCache coreCache

// InitCoreCache 初始化核心缓存
func InitCoreCache(configMap map[string]config.Config) {
	c := &cache{
		models:         &sync.Map{},
		devices:        &sync.Map{},
		points:         &sync.Map{},
		runningPlugins: &sync.Map{},
		modelPlugin:    &sync.Map{},
	}

	for key, _ := range configMap {
		for _, model := range configMap[key].DeviceModels {
			c.models.Store(model.Name, model)    // cache model
			c.modelPlugin.Store(model.Name, key) // cache modelPlugin

			for _, pointMap := range model.DevicePoints {
				point := pointMap.ToPoint()
				c.points.Store(model.Name+"_"+point.Name, point) // cache point
			}

			for _, device := range model.Devices {
				device.ModelName = model.Name        // 覆写模型名称
				c.devices.Store(device.Name, device) // cache device
			}
		}
	}

	CoreCache = c
}

// coreCache 核心缓存
type coreCache interface {
	GetModel(modelName string) (model config.DeviceModel, ok bool)                      // model info
	GetDevice(deviceName string) (device config.Device, ok bool)                        // device info
	GetPointByModel(modelName string, pointName string) (point config.Point, ok bool)   // search point by model
	GetPointByDevice(deviceName string, pointName string) (point config.Point, ok bool) // search point by device
	GetRunningPluginByModel(modelName string) (plugin contracts.Plugin, ok bool)        // search plugin by model
	GetRunningPluginByDevice(deviceName string) (plugin contracts.Plugin, ok bool)      // search plugin by device
	GetRunningPluginByKey(key string) (plugin contracts.Plugin, ok bool)                // search plugin by directory name
	AddRunningPlugin(key string, plugin contracts.Plugin)                               // add running plugin
	Models() (models []config.DeviceModel)                                              // all model
	Devices() (devices []config.Device)                                                 // all device
}

type cache struct {
	models         *sync.Map // device model cache, modelName => config.DeviceModel
	devices        *sync.Map // device cache, deviceName => config.Device
	points         *sync.Map // point cache, modelName_pointName => config.Point
	runningPlugins *sync.Map // running plugin cache, id => contracts.Plugin, id is core config directory name
	modelPlugin    *sync.Map // model name => plugin key, modelName => id
}

func (c *cache) GetModel(modelName string) (model config.DeviceModel, ok bool) {
	if raw, exist := c.models.Load(modelName); exist {
		model, _ = raw.(config.DeviceModel)
		return model, true
	}
	return config.DeviceModel{}, false
}

func (c *cache) GetDevice(deviceName string) (device config.Device, ok bool) {
	if raw, exist := c.devices.Load(deviceName); exist {
		device, _ = raw.(config.Device)
		return device, true
	}
	return config.Device{}, false
}

func (c *cache) GetPointByModel(modelName string, pointName string) (point config.Point, ok bool) {
	key := modelName + "_" + pointName
	if raw, ok := c.points.Load(key); ok {
		point, _ = raw.(config.Point)
		return point, true
	}
	return config.Point{}, false
}

func (c *cache) GetPointByDevice(deviceName string, pointName string) (point config.Point, ok bool) {
	device, ok := c.GetDevice(deviceName)
	if !ok {
		return config.Point{}, false
	}
	return c.GetPointByModel(device.ModelName, pointName)
}

func (c *cache) GetRunningPluginByModel(modelName string) (plugin contracts.Plugin, ok bool) {
	if raw, ok := c.modelPlugin.Load(modelName); ok {
		key, _ := raw.(string)
		return c.GetRunningPluginByKey(key)
	}
	return nil, false
}

func (c *cache) GetRunningPluginByDevice(deviceName string) (plugin contracts.Plugin, ok bool) {
	device, ok := c.GetDevice(deviceName)
	if !ok {
		return nil, false
	}
	return c.GetRunningPluginByModel(device.ModelName)
}

func (c *cache) GetRunningPluginByKey(key string) (plugin contracts.Plugin, ok bool) {
	if raw, ok := c.runningPlugins.Load(key); ok {
		plugin, _ = raw.(contracts.Plugin)
		return plugin, true
	}
	return nil, false
}

func (c *cache) AddRunningPlugin(key string, plugin contracts.Plugin) {
	c.runningPlugins.Store(key, plugin)
}

func (c *cache) Models() (models []config.DeviceModel) {
	c.models.Range(func(key, value any) bool {
		model, _ := value.(config.DeviceModel)
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
