package cmanager

import (
	"encoding/json"
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"os"
	"path/filepath"
	"sort"
)

var Manager *manager

type manager struct {
	path        string   // 驱动配置路径
	folderNames []string // 用于固定遍历顺序
	configs     map[string]config.Config
}

// Load 从指定路径加载所有配置
func Load(path string) error {
	// 加载所有配置
	configMap, err := parseFromPath(path)
	if err != nil {
		return err
	}

	// 获取配置目录名称
	var folderNames []string
	for k, _ := range configMap {
		folderNames = append(folderNames, k)
	}
	// 排序
	sort.Strings(folderNames)

	Manager = &manager{
		path:        path,
		folderNames: folderNames,
		configs:     configMap,
	}

	return nil
}

func (m *manager) GetConfigMap() map[string]config.Config {
	return m.configs
}

// MergeConfig 合并配置
// 以最新数据为准，追加、覆盖原始数据
func (m *manager) MergeConfig(c config.Config) error {
	folderName := fmt.Sprintf("gen-%s", c.ProtocolName)
	// 是否存在相同协议配置文件
	// 存在：覆盖合并
	// 不存在：新增
	for _, name := range m.folderNames {
		if m.configs[name].ProtocolName == c.ProtocolName {
			// 覆盖合并
			// 1. 合并模型
			modelMap := m.mergeModels(m.configs[name].DeviceModels, c.DeviceModels)
			// 2. 合并设备
			deviceMap := m.mergeDevices(m.getAllDevice(m.configs[name]), m.getAllDevice(c))
			// 2.1 添加设备到模型下
			for s, model := range modelMap {
				model.Devices = deviceMap[s]
				modelMap[s] = model
			}
			// 3. 合并连接
			connections := m.mergeConnections(m.configs[name].Connections, c.Connections)
			// 4. 组合
			models := make([]config.DeviceModel, 0)
			for k, _ := range modelMap {
				models = append(models, modelMap[k])
			}
			c = config.Config{
				DeviceModels: models,
				Connections:  connections,
				ProtocolName: c.ProtocolName,
				Tasks:        make([]config.TimerTask, 0),
			}
			folderName = name
		}
	}

	// 保存
	return m.save(folderName, c)
}

// UpdateDeviceArea 更新设备区域信息
func (m *manager) UpdateDeviceArea(sn string, area string) error {
	// 所有配置
	for name, _ := range m.configs {
		// 所有模型
		for i, model := range m.configs[name].DeviceModels {
			// 模型下所有设备
			for j, device := range model.Devices {
				if device.DeviceSn == sn {
					// 防止空属性异常
					if device.Properties == nil {
						device.Properties = make(map[string]string)
					}
					// 更新区域信息
					device.Properties["area"] = area
					m.configs[name].DeviceModels[i].Devices[j] = device
					// 持久化到文件
					return m.save(name, m.configs[name])
				}
			}
		}
	}
	return nil
}

// getAllDevice 获取指定配置下所有设备信息
func (m *manager) getAllDevice(c config.Config) []config.Device {
	var devices []config.Device
	for _, model := range c.DeviceModels {
		for _, device := range model.Devices {
			device.ModelName = model.Name
			devices = append(devices, device)
		}
	}
	return devices
}

// mergeModels 合并模型
// 注意：需要清除模型下关联设备
func (m *manager) mergeModels(old []config.DeviceModel, new []config.DeviceModel) map[string]config.DeviceModel {
	mergeModels := make(map[string]config.DeviceModel)
	for _, model := range old {
		// 移除模型下所有设备
		model.Devices = make([]config.Device, 0)
		// 通过模型名称区分（模型名称不同，物模型ID可能相同）
		mergeModels[model.Name] = model
	}
	// 合并模型
	for _, model := range new {
		// 移除模型下所有设备
		model.Devices = make([]config.Device, 0)
		mergeModels[model.Name] = model
	}
	return mergeModels
}

// mergeDevices 合并设备信息
// 注意：入参 config.Device 需要携带模型信息
func (m *manager) mergeDevices(old []config.Device, new []config.Device) map[string][]config.Device {
	// 合并设备
	deviceMap := make(map[string]config.Device)
	for _, device := range old {
		deviceMap[device.DeviceSn] = device
	}
	for _, device := range new {
		deviceMap[device.DeviceSn] = device
	}

	// 转换数据，以模型名称作为设备列表索引
	mergeDevices := make(map[string][]config.Device)
	for _, device := range deviceMap {
		if _, exist := mergeDevices[device.ModelName]; !exist {
			mergeDevices[device.ModelName] = make([]config.Device, 0)
		}
		mergeDevices[device.ModelName] = append(mergeDevices[device.ModelName], device)
	}

	return mergeDevices
}

// mergeConnections 合并连接
func (m *manager) mergeConnections(old map[string]any, new map[string]any) map[string]any {
	mergeConnections := make(map[string]any)
	for k, _ := range old {
		mergeConnections[k] = old[k]
	}
	for k, _ := range new {
		mergeConnections[k] = new[k]
	}
	return mergeConnections
}

// save 持久化数据到配置文件
func (m *manager) save(folderName string, c config.Config) error {
	// 创建文件夹
	dir := filepath.Join(m.path, folderName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	file := filepath.Join(dir, "config.json")
	b, _ := json.MarshalIndent(c, "", "\t")
	return os.WriteFile(file, b, 0666)
}
