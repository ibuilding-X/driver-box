// 缓存助手

package helper

import (
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/helper/cmanager"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"go.uber.org/zap"
	"sync"
)

const (
	businessPropSN       string = "_sn"
	businessPropParentID string = "_parentID"
	businessPropSystemID string = "_systemID"
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
		tagDevices:         &sync.Map{},
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
				checkPoint(&deviceModel, &point)
				pointMap[point.Name] = point
				if _, ok := model.Points[point.Name]; !ok {
					model.Points[point.Name] = point
				}
			}
			for _, device := range deviceModel.Devices {
				if device.ID == "" {
					Logger.Error("config error , device id is empty", zap.Any("device", device))
					continue
				}
				deviceId := device.ID
				device.ModelName = deviceModel.Name
				if _, ok := model.Devices[deviceId]; !ok {
					model.Devices[deviceId] = device
				}
				if deviceRaw, ok := c.devices.Load(deviceId); !ok {
					c.devices.Store(deviceId, device)
				} else {
					storedDeviceBase := deviceRaw.(config.Device)
					if storedDeviceBase.ModelName != device.ModelName {
						return fmt.Errorf("conflict model for device [%s]: %s -> %s", device.ID,
							device.ModelName, storedDeviceBase.ModelName)
					}
				}
				var properties map[string]DeviceProperties
				if raw, ok := c.deviceProperties.Load(deviceId); ok {
					properties = raw.(map[string]DeviceProperties)
				} else {
					properties = make(map[string]DeviceProperties)
				}
				properties[configMap[key].ProtocolName+"_"+device.ConnectionKey] = device.Properties
				c.deviceProperties.Store(deviceId, properties)
				for _, point := range pointMap {
					devicePointKey := deviceId + "_" + point.Name
					if _, ok := c.points.Load(devicePointKey); ok {
						return fmt.Errorf("device %s duplicate point %s found", deviceId, point.Name)
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

// 检查点位配置合法性
func checkPoint(model *config.DeviceModel, point *config.Point) {
	if point.Name == "" {
		Logger.Error("config error , point name is empty", zap.Any("point", point), zap.String("modelName", model.Name))
	}
	if point.Description == "" {
		Logger.Warn("config error , point description is empty", zap.Any("point", point), zap.String("model", model.Name))
	}
	if point.ValueType != config.ValueType_Float && point.ValueType != config.ValueType_Int && point.ValueType != config.ValueType_String {
		Logger.Error("point valueType config error , valid config is: int float string", zap.Any("point", point), zap.String("model", model.Name))
	}
	if point.ReportMode == "" {
		Logger.Warn("config error , point reportMode is empty, set default to real")
		point.ReportMode = config.ReportMode_Real
	}
	if point.ReadWrite != config.ReadWrite_RW && point.ReadWrite != config.ReadWrite_R && point.ReadWrite != config.ReadWrite_W {
		Logger.Error("point readWrite config error , valid config is: R W RW", zap.Any("point", point), zap.String("model", model.Name))
	}
	if point.ReportMode != config.ReportMode_Real && point.ReportMode != config.ReportMode_Change {
		Logger.Error("point reportMode config error , valid config is: realTime change period", zap.Any("point", point), zap.String("model", model.Name))
	}
	//存在精度换算时，点位类型要求float
	if point.Scale != 0 && point.ValueType != config.ValueType_Float {
		Logger.Error("point scale config error , valid config is: float", zap.Any("point", point), zap.String("model", model.Name))
	}
}

// coreCache 核心缓存
type coreCache interface {
	GetModel(modelName string) (model config.Model, ok bool)                               // model info
	GetDevice(id string) (device config.Device, ok bool)                                   // device info
	GetDeviceByDeviceAndPoint(id string, pointName string) (device config.Device, ok bool) // connection config
	//查询指定标签的设备列表
	GetDevicesByTag(tag string) (devices []config.Device)
	AddTag(tag string) (e error)                                                                  //
	GetPointByModel(modelName string, pointName string) (point config.Point, ok bool)             // search point by model
	GetPointByDevice(id string, pointName string) (point config.Point, ok bool)                   // search point by device
	GetRunningPluginByDeviceAndPoint(id string, pointName string) (plugin plugin.Plugin, ok bool) // search plugin by device and point
	GetRunningPluginByKey(key string) (plugin plugin.Plugin, ok bool)                             // search plugin by directory name
	AddRunningPlugin(key string, plugin plugin.Plugin)                                            // add running plugin
	Models() (models []config.Model)                                                              // all model
	Devices() (devices []config.Device)
	GetProtocolsByDevice(id string) (map[string]DeviceProperties, bool) // device protocols
	GetAllRunningPluginKey() (keys []string)                            // get running plugin keys
	UpdateDeviceProperty(id string, key string, value string)           // 更新设备属性
	DeleteDevice(id string)                                             // 删除设备
	UpdateDeviceDesc(id string, desc string)                            // 更新设备描述
	Reset()

	// businessPropCache 业务属性接口
	businessPropCache

	// configManager 配置管理器接口
	configManager
}

// businessPropCache 业务属性缓存
type businessPropCache interface {
	GetDeviceBusinessProp(id string) (props config.DeviceBusinessProp, err error) // 获取设备业务属性
	UpdateDeviceBusinessPropSN(id string, value string) error                     // 更新设备业务属性SN
	UpdateDeviceBusinessPropParentID(id string, value string) error               // 更新设备业务属性ParentID
	UpdateDeviceBusinessPropSystemID(sn string, value string) error               // 更新设备业务属性SystemID
}

// configManager 配置管理器接口
type configManager interface {
	// AddConnection 新增连接
	AddConnection(plugin string, key string, conn any) error
	// GetConnection 获取连接信息
	GetConnection(key string) (any, error)
	// GetConnectionPluginName 获取连接所属的插件名称
	GetConnectionPluginName(key string) string
	// AddModel 新增模型
	AddModel(plugin string, model config.DeviceModel) error
	// AddOrUpdateDevice 新增或更新设备
	AddOrUpdateDevice(device config.Device) error
	// RemoveDevice 删除设备
	RemoveDevice(modelName string, deviceID string) error
	// RemoveDeviceByID 根据 ID 删除设备
	RemoveDeviceByID(id string) error
	// BatchRemoveDevice 批量删除设备
	BatchRemoveDevice(ids []string) error
}

func (c *cache) GetModel(modelName string) (model config.Model, ok bool) {
	if raw, exist := c.models.Load(modelName); exist {
		m, _ := raw.(config.Model)
		return m, true
	}
	return config.Model{}, false
}

func (c *cache) GetDevice(id string) (device config.Device, ok bool) {
	if raw, exist := c.devices.Load(id); exist {
		device, _ = raw.(config.Device)
		return device, true
	}
	return config.Device{}, false
}

func (c *cache) GetDeviceByDeviceAndConn(id, connectionKey string) (device config.Device, ok bool) {
	if raw, ok := c.deviceProperties.Load(id + "_" + connectionKey); ok {
		device, _ = raw.(config.Device)
		return device, true
	}
	return config.Device{}, false
}

func (c *cache) GetDeviceByDeviceAndPoint(id, pointName string) (device config.Device, ok bool) {
	if raw, ok := c.devicePointConn.Load(id + "_" + pointName); ok {
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

// GetPointByDevice 查询指定设备的指定点位信息
func (c *cache) GetPointByDevice(id string, pointName string) (point config.Point, ok bool) {
	// 查询设备
	if device, ok := c.GetDevice(id); ok {
		return c.GetPointByModel(device.ModelName, pointName)
	}
	return config.Point{}, false
}

func (c *cache) GetRunningPluginByDeviceAndPoint(id, pointName string) (plugin plugin.Plugin, ok bool) {
	if key, ok := c.devicePointPlugins.Load(fmt.Sprintf("%s_%s", id, pointName)); ok {
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

func (c *cache) GetProtocolsByDevice(id string) (map[string]DeviceProperties, bool) {
	if raw, ok := c.deviceProperties.Load(id); ok {
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

// UpdateDeviceProperty 更新设备属性并持久化
func (c *cache) UpdateDeviceProperty(id string, key string, value string) {
	_ = c.updateDeviceProp(id, key, value)
}

// DeleteDevice 删除设备
func (c *cache) DeleteDevice(id string) {
	c.devices.Delete(id)
}

// UpdateDeviceDesc 更新设备描述
func (c *cache) UpdateDeviceDesc(id string, desc string) {
	if deviceAny, ok := c.devices.Load(id); ok {
		device, _ := deviceAny.(config.Device)
		device.Description = desc
		c.devices.Store(id, device)
	}
}

func (c *cache) resetSyncMap(m *sync.Map) {
	m.Range(func(key, value any) bool {
		m.Delete(key)
		return true
	})
}

// Reset 重置数据
func (c *cache) Reset() {
	c.resetSyncMap(c.models)
	c.resetSyncMap(c.devices)
	c.resetSyncMap(c.deviceProperties)
	c.resetSyncMap(c.points)
	c.resetSyncMap(c.devicePointPlugins)
	c.resetSyncMap(c.runningPlugins)
	c.resetSyncMap(c.devicePointConn)
	c.resetSyncMap(c.tagDevices)
}

// AddOrUpdateDevice 添加或更新设备
func (c *cache) AddOrUpdateDevice(device config.Device) error {
	// 自动补全设备描述
	if device.Description == "" {
		device.Description = device.ID
	}
	// 更新缓存信息
	c.devices.Store(device.ID, device)
	// 持久化
	return cmanager.AddOrUpdateDevice(device)
}

// GetDeviceBusinessProp 获取设备业务属性
func (c *cache) GetDeviceBusinessProp(id string) (props config.DeviceBusinessProp, err error) {
	if raw, ok := c.devices.Load(id); ok {
		device, _ := raw.(config.Device)
		return config.DeviceBusinessProp{
			SN:       device.Properties[businessPropSN],
			ParentID: device.Properties[businessPropParentID],
			SystemID: device.Properties[businessPropSystemID],
		}, nil
	}
	return config.DeviceBusinessProp{}, fmt.Errorf("device %s not found", id)
}

// updateDeviceProp 更新设备属性
func (c *cache) updateDeviceProp(id, key, value string) error {
	if deviceAny, ok := c.devices.Load(id); ok {
		device, _ := deviceAny.(config.Device)
		if device.Properties == nil {
			device.Properties = make(map[string]string)
		}
		device.Properties[key] = value
		// 更新缓存
		c.devices.Store(id, device)
		// 持久化
		return cmanager.AddOrUpdateDevice(device)
	}
	return fmt.Errorf("device %s not found", id)
}

// UpdateDeviceBusinessPropSN 更新设备业务属性SN
func (c *cache) UpdateDeviceBusinessPropSN(id string, value string) error {
	return c.updateDeviceProp(id, businessPropSN, value)
}

// UpdateDeviceBusinessPropParentID 更新设备业务属性ParentID
func (c *cache) UpdateDeviceBusinessPropParentID(id string, value string) error {
	return c.updateDeviceProp(id, businessPropParentID, value)
}

// UpdateDeviceBusinessPropSystemID 更新设备业务属性SystemID
func (c *cache) UpdateDeviceBusinessPropSystemID(sn string, value string) error {
	return c.updateDeviceProp(sn, businessPropSystemID, value)
}

// AddConnection 新增连接
func (c *cache) AddConnection(plugin string, key string, conn any) error {
	return cmanager.AddConnection(plugin, key, conn)
}

// GetConnection 获取连接信息
func (c *cache) GetConnection(key string) (any, error) {
	return cmanager.GetConnection(key)
}

// GetConnectionPluginName 获取连接所属的插件名称
func (c *cache) GetConnectionPluginName(key string) string {
	return cmanager.GetConnectionPluginName(key)
}

// AddModel 新增模型
func (c *cache) AddModel(plugin string, model config.DeviceModel) error {
	return cmanager.AddModel(plugin, model)
}

// RemoveDevice 根据 ID 删除设备
func (c *cache) RemoveDevice(modelName string, deviceID string) error {
	return cmanager.RemoveDevice(modelName, deviceID)
}

// RemoveDeviceByID 根据 ID 删除设备
func (c *cache) RemoveDeviceByID(id string) error {
	return cmanager.RemoveDeviceByID(id)
}

// BatchRemoveDevice 批量删除设备
func (c *cache) BatchRemoveDevice(ids []string) error {
	return cmanager.BatchRemoveDevice(ids)
}
