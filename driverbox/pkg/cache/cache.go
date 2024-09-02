// 缓存助手

package cache

import (
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
)

type DeviceProperties map[string]string

// coreCache 核心缓存
type CoreCache interface {
	GetModel(modelName string) (model config.Model, ok bool) // model info
	GetDevice(id string) (device config.Device, ok bool)
	//查询指定标签的设备列表
	GetDevicesByTag(tag string) (devices []config.Device)
	AddTag(tag string) (e error)                                                      //
	GetPointByModel(modelName string, pointName string) (point config.Point, ok bool) // search point by model
	GetPointByDevice(id string, pointName string) (point config.Point, ok bool)       // search point by device
	GetRunningPluginByDevice(deviceId string) (plugin plugin.Plugin, ok bool)         // search plugin by device and point
	GetRunningPluginByKey(key string) (plugin plugin.Plugin, ok bool)                 // search plugin by directory name
	AddRunningPlugin(key string, plugin plugin.Plugin)                                // add running plugin
	Models() (models []config.Model)                                                  // all model
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
