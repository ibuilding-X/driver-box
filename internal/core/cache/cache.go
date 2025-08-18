package cache

import (
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/event"
	"github.com/ibuilding-x/driver-box/driverbox/helper/cmanager"
	cache2 "github.com/ibuilding-x/driver-box/driverbox/pkg/cache"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/internal/core/shadow"
	"github.com/ibuilding-x/driver-box/internal/export"
	"github.com/ibuilding-x/driver-box/internal/logger"
	"go.uber.org/zap"
	"sync"
)

const (
	businessPropSN       string = "_sn"
	businessPropParentID string = "_parentID"
	businessPropSystemID string = "_systemID"
)

var Instance cache2.CoreCache

type cache struct {
	models         *sync.Map // name => config.Model
	devices        *sync.Map // deviceSn => config.Device
	devicePlugins  *sync.Map // deviceSn => plugin key
	runningPlugins *sync.Map // key => plugin.Plugin
	//根据tag分组存储的设备列表
	tagDevices *sync.Map // tag => []config.Device
}

// InitCoreCache 初始化核心缓存
func InitCoreCache(configMap map[string]config.Config) (obj cache2.CoreCache, err error) {
	c := &cache{
		models:         &sync.Map{},
		devices:        &sync.Map{},
		devicePlugins:  &sync.Map{},
		runningPlugins: &sync.Map{},
		tagDevices:     &sync.Map{},
	}
	Instance = c

	for key, _ := range configMap {
		for _, deviceModel := range configMap[key].DeviceModels {
			var model config.Model
			if raw, ok := c.models.Load(deviceModel.Name); ok {
				model = raw.(config.Model)
				if model.ModelBase.Name != deviceModel.ModelBase.Name ||
					model.ModelBase.ModelID != deviceModel.ModelBase.ModelID {
					return c, fmt.Errorf("conflict model base information: %v  %v",
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
					logger.Logger.Error("config error , device id is empty", zap.Any("device", device))
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
						return c, fmt.Errorf("conflict model for device [%s]: %s -> %s", device.ID,
							device.ModelName, storedDeviceBase.ModelName)
					}
				}
				c.devicePlugins.Store(deviceId, key)

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
	return c, nil
}

// 检查点位配置合法性
func checkPoint(model *config.DeviceModel, point *config.Point) {
	if point.Name == "" {
		logger.Logger.Error("config error , point name is empty", zap.Any("point", point), zap.String("modelName", model.Name))
	}
	if point.Description == "" {
		logger.Logger.Warn("config error , point description is empty", zap.Any("point", point), zap.String("model", model.Name))
	}
	if point.ValueType != config.ValueType_Float && point.ValueType != config.ValueType_Int && point.ValueType != config.ValueType_String {
		logger.Logger.Error("point valueType config error , valid config is: int float string", zap.Any("point", point), zap.String("model", model.Name))
	}
	if point.ReportMode == "" {
		logger.Logger.Warn("config error , point reportMode is empty, set default to real")
		point.ReportMode = config.ReportMode_Real
	}
	if point.ReadWrite != config.ReadWrite_RW && point.ReadWrite != config.ReadWrite_R && point.ReadWrite != config.ReadWrite_W {
		logger.Logger.Error("point readWrite config error , valid config is: R W RW", zap.Any("point", point), zap.String("model", model.Name))
	}
	if point.ReportMode != config.ReportMode_Real && point.ReportMode != config.ReportMode_Change {
		logger.Logger.Error("point reportMode config error , valid config is: realTime change period", zap.Any("point", point), zap.String("model", model.Name))
	}
	//存在精度换算时，点位类型要求float
	if point.Scale != 0 && point.ValueType != config.ValueType_Float {
		logger.Logger.Error("point scale config error , valid config is: float", zap.Any("point", point), zap.String("model", model.Name))
	}
}
func (c *cache) AddTag(tag string) (e error) {
	//TODO implement me
	panic("implement me")
}
func (c *cache) GetModel(modelName string) (model config.Model, ok bool) {
	if raw, exist := c.models.Load(modelName); exist {
		m, _ := raw.(config.Model)
		return m, true
	}
	return config.Model{}, false
}

func (c *cache) GetPoints(modelName string) ([]config.Point, bool) {
	points := make([]config.Point, 0)
	if model, exist := cmanager.GetModel(modelName); exist {
		for _, point := range model.DevicePoints {
			points = append(points, point.ToPoint())
		}
		return points, true
	}
	return points, false
}
func (c *cache) GetDevice(id string) (device config.Device, ok bool) {
	if raw, exist := c.devices.Load(id); exist {
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

func (c *cache) GetRunningPluginByDevice(id string) (plugin plugin.Plugin, ok bool) {
	if key, ok := c.devicePlugins.Load(id); ok {
		return c.GetRunningPluginByKey(key.(string))
	}
	logger.Logger.Error("device not found plugin", zap.String("id", id))
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
	e := c.BatchRemoveDevice([]string{id})
	if e != nil {
		logger.Logger.Error("remove device error", zap.String("id", id))
	}
}

// UpdateDeviceDesc 更新设备描述
func (c *cache) UpdateDeviceDesc(id string, desc string) {
	if deviceAny, ok := c.devices.Load(id); ok {
		device, _ := deviceAny.(config.Device)
		device.Description = desc
		c.devices.Store(id, device)
		// 持久化
		_ = cmanager.AddOrUpdateDevice(device)
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
	c.resetSyncMap(c.devicePlugins)
	c.resetSyncMap(c.runningPlugins)
	c.resetSyncMap(c.tagDevices)
}

// AddOrUpdateDevice 添加或更新设备
// 更新内容列表
// * 核心缓存设备
// * 设备影子
// * 持久化文件
func (c *cache) AddOrUpdateDevice(device config.Device) error {
	if logger.Logger != nil {
		logger.Logger.Info("core cache add device", zap.Any("device", device), zap.Any("model", device.ModelName))
	}
	// 查找模型信息
	model, ok := cmanager.GetModel(device.ModelName)
	if !ok {
		logger.Logger.Error("model not found", zap.String("modelName", device.ModelName))
		return fmt.Errorf("model %s not found", device.ModelName)
	}
	// 校验设备是否已存在
	deviceRaw, ok := c.devices.Load(device.ID)
	if ok {
		storedDeviceBase := deviceRaw.(config.Device)
		if storedDeviceBase.ModelName != device.ModelName {
			logger.Logger.Error("conflict model for device", zap.String("deviceId", device.ID))
			return fmt.Errorf("conflict model for device [%s]: %s -> %s", device.ID,
				device.ModelName, storedDeviceBase.ModelName)
		}
	}

	// 查找配置 key
	key := cmanager.GetConfigKeyByModel(model.Name)
	if key == "" {
		logger.Logger.Error("config key not found", zap.String("modelName", model.Name))
		return fmt.Errorf("config key not found for model %s", model.Name)
	}
	// 更新 devicePlugins
	c.devicePlugins.Store(device.ID, key)
	// 自动补全设备描述
	if device.Description == "" {
		device.Description = device.ID
	}
	// 更新核心缓存
	_, ok = c.devices.Load(device.ID)
	if !ok {
		defer export.TriggerEvents(event.EventCodeAddDevice, device.ID, nil)
	}
	c.devices.Store(device.ID, device)
	// 更新设备影子
	if shadow.DeviceShadow != nil && !shadow.DeviceShadow.HasDevice(device.ID) {
		shadow.DeviceShadow.AddDevice(device.ID, device.ModelName)
	}
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
	return cmanager.GetPluginNameByConnection(key)
}

// AddModel 新增模型
func (c *cache) AddModel(plugin string, model config.DeviceModel) error {
	err := cmanager.AddModel(plugin, model)
	if err == nil {
		// 判断模型是否存在
		if _, ok := c.models.Load(model.Name); ok {
			return nil
		}
		// 添加模型
		c.models.Store(model.Name, model.ToModel())
	}
	return err
}

// BatchRemoveDevice 批量删除设备
func (c *cache) BatchRemoveDevice(ids []string) error {
	for _, id := range ids {
		export.TriggerEvents(event.EventCodeWillDeleteDevice, id, nil)
		c.devices.Delete(id)
	}
	// 删除设备影子
	if shadow.DeviceShadow != nil {
		_ = shadow.DeviceShadow.DeleteDevice(ids...)
	}
	return cmanager.BatchRemoveDevice(ids)
}
