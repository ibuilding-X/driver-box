package cmanager

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/internal/logger"
	"go.uber.org/zap"
)

const (
	// DefaultConfigPath is the default path to the config directory.
	DefaultConfigPath string = "./res/driver"
	// DefaultConfigName is the default name of the config file.
	DefaultConfigName string = "config.json"
	// DefaultScriptName is the default name of the script file.
	DefaultScriptName string = "converter.lua"
)

var (
	// ErrDeviceModelNameNull is the error returned when the device model name is empty.
	ErrDeviceModelNameNull = errors.New("device model name is null")
	// ErrUnknownModel is the error returned when the model is unknown.
	ErrUnknownModel = errors.New("unknown model")
	// ErrUnknownDevice is the error returned when the device is unknown.
	ErrUnknownDevice = errors.New("unknown device")
	// ErrConfigNotExist is the error returned when the config file does not exist.
	ErrConfigNotExist = errors.New("config not exist")
	// ErrConfigEmpty is the error returned when the config file is empty.
	ErrConfigEmpty = errors.New("config is empty")
	// ErrUnknownPlugin is the error returned when the plugin is unknown.
	ErrUnknownPlugin = errors.New("unknown plugin")
)

// instance 全局配置管理器实例
var instance = New()

// Manager 配置管理器接口
type Manager interface {
	// -------------------- 配置相关 --------------------

	// SetConfigPath 设置配置目录
	SetConfigPath(path string)
	// SetConfigFileName 设置配置文件名
	SetConfigFileName(name string)
	// SetScriptFileName 设置脚本文件名
	SetScriptFileName(name string)
	// LoadConfig 加载配置
	LoadConfig() error
	// GetConfigs 获取所有配置
	GetConfigs() map[string]config.Config
	// AddConfig 新增配置
	AddConfig(c config.Config) error
	// GetConfig 根据 key 获取配置
	GetConfig(key string) (config.Config, bool)
	// GetConfigKeyByModel 根据模型名获取配置 key
	GetConfigKeyByModel(modelName string) string
	// OptimizeConfig 优化所有驱动配置（移除未使用模型、移除未使用连接）
	// 提示：此操作会修改并持久化驱动配置文件
	OptimizeConfig()

	// -------------------- 模型相关 --------------------

	// GetModel 获取模型
	GetModel(modelName string) (config.DeviceModel, bool)
	// AddModel 新增模型
	AddModel(plugin string, model config.DeviceModel) error

	// -------------------- 设备相关 --------------------

	// GetDevice 获取设备
	GetDevice(modelName string, deviceID string) (config.Device, bool)
	// AddOrUpdateDevice 新增或更新设备
	AddOrUpdateDevice(device config.Device) error
	// RemoveDevice 删除设备
	RemoveDevice(modelName string, deviceID string) error
	// RemoveDeviceByID 根据 ID 删除设备
	RemoveDeviceByID(id string) error
	// BatchRemoveDevice 批量删除设备
	BatchRemoveDevice(ids []string) error

	// -------------------- 连接相关 --------------------

	// AddConnection 新增连接
	AddConnection(plugin string, key string, conn any) error
	// GetConnection 获取连接信息
	// Deprecated: 当不同协议的连接 Key 相同时，可能会导致获取的连接信息不准确
	GetConnection(key string) (any, error)
	// RemoveConnection 删除连接
	// Deprecated: 当不同协议的连接 Key 相同时，可能会导致误删除
	RemoveConnection(key string) error
	// GetConnections 获取指定插件下所有连接配置
	GetConnections(plugin string) (map[string]interface{}, error)

	// -------------------- 插件相关 --------------------

	// GetPluginNameByModel 获取模型所属的插件名称
	GetPluginNameByModel(modelName string) string
	// GetPluginNameByConnection 获取连接所属的插件名称
	GetPluginNameByConnection(key string) string
}

// manager 实现配置管理器接口
type manager struct {
	root       string
	configName string
	scriptName string
	configs    map[string]config.Config
	mux        *sync.RWMutex
	reload     atomic.Bool
}

// New 创建配置管理器实例
func New() Manager {
	return &manager{
		root:       DefaultConfigPath,
		configName: DefaultConfigName,
		scriptName: DefaultScriptName,
		configs:    make(map[string]config.Config),
		mux:        &sync.RWMutex{},
	}
}

func SetConfigPath(path string) {
	instance.SetConfigPath(path)
}

func SetConfigFileName(name string) {
	instance.SetConfigFileName(name)
}

func SetScriptFileName(name string) {
	instance.SetScriptFileName(name)
}

func LoadConfig() error {
	return instance.LoadConfig()
}

func GetConfigs() map[string]config.Config {
	return instance.GetConfigs()
}

func AddConfig(c config.Config) error {
	return instance.AddConfig(c)
}

func GetConfig(key string) (config.Config, bool) {
	return instance.GetConfig(key)
}

func GetConfigKeyByModel(modelName string) string {
	return instance.GetConfigKeyByModel(modelName)
}

func OptimizeConfig() {
	instance.OptimizeConfig()
}

func GetModel(modelName string) (config.DeviceModel, bool) {
	return instance.GetModel(modelName)
}

func AddModel(plugin string, model config.DeviceModel) error {
	return instance.AddModel(plugin, model)
}

func GetDevice(modelName string, deviceID string) (config.Device, bool) {
	return instance.GetDevice(modelName, deviceID)
}

func AddOrUpdateDevice(device config.Device) error {
	return instance.AddOrUpdateDevice(device)
}

func RemoveDevice(modelName string, deviceID string) error {
	return instance.RemoveDevice(modelName, deviceID)
}

func RemoveDeviceByID(id string) error {
	return instance.RemoveDeviceByID(id)
}

func BatchRemoveDevice(ids []string) error {
	return instance.BatchRemoveDevice(ids)
}

func AddConnection(plugin string, key string, conn any) error {
	return instance.AddConnection(plugin, key, conn)
}

func GetConnection(key string) (any, error) {
	return instance.GetConnection(key)
}

func RemoveConnection(key string) error {
	return instance.RemoveConnection(key)
}

func GetConnections(plugin string) (map[string]interface{}, error) {
	return instance.GetConnections(plugin)
}

func GetPluginNameByModel(modelName string) string {
	return instance.GetPluginNameByModel(modelName)
}

func GetPluginNameByConnection(key string) string {
	return instance.GetPluginNameByConnection(key)
}

// SetConfigPath 设置配置目录
func (m *manager) SetConfigPath(path string) {
	m.mux.Lock()
	defer m.mux.Unlock()

	if path != "" {
		m.root = path
	}
}

// SetConfigFileName 设置配置文件名
func (m *manager) SetConfigFileName(name string) {
	m.mux.Lock()
	defer m.mux.Unlock()

	if name != "" {
		m.configName = name
	}
}

// SetScriptFileName 设置脚本文件名
func (m *manager) SetScriptFileName(name string) {
	m.mux.Lock()
	defer m.mux.Unlock()

	if name != "" {
		m.scriptName = name
	}
}

// LoadConfig 加载配置
func (m *manager) LoadConfig() error {
	m.mux.Lock()
	defer m.mux.Unlock()
	defer m.reload.Store(true)

	// 自动创建配置目录
	if err := m.createDir(m.root); err != nil {
		return err
	}

	// 遍历配置目录，获取所有文件夹
	dirs := m.getSubDirs()
	if len(dirs) <= 0 {
		return nil
	}

	//config 协议唯一性
	protocols := make(map[string]string)

	// 解析每个文件夹的配置
	for i, _ := range dirs {
		path := filepath.Join(m.root, dirs[i], m.configName)
		c, err := m.parseConfigFromFile(path)
		if err != nil {
			if errors.Is(err, ErrConfigNotExist) {
				continue
			}
			if errors.Is(err, ErrConfigEmpty) {
				continue
			}
			logger.Logger.Error("parse config from file error", zap.String("path", path), zap.Error(err))
			return err
		}
		// fix：填充配置文件 Key 字段
		c.Key = dirs[i]
		if preConfig, ok := protocols[c.ProtocolName]; ok {
			return fmt.Errorf("protocol:%s is repeated, prePath: %s, curPath: %s", c.ProtocolName, preConfig, path)
		}
		protocols[c.ProtocolName] = path

		// 保存配置
		m.configs[dirs[i]] = c.UpdateIndexAndClean()
	}

	if m.reload.Load() {
		return nil
	}

	// 优化配置文件
	m.optimizeConfig()

	// 保存
	for k, _ := range m.configs {
		if err := m.saveConfig(k); err != nil {
			return err
		}
	}

	return nil
}

// GetConfigs 获取所有配置
func (m *manager) GetConfigs() map[string]config.Config {
	m.mux.RLock()
	defer m.mux.RUnlock()

	return m.configs
}

// GetConfig 根据 key 获取配置
func (m *manager) GetConfig(key string) (config.Config, bool) {
	m.mux.RLock()
	defer m.mux.RUnlock()

	if c, ok := m.configs[key]; ok {
		return c, true
	}
	return config.Config{}, false
}

// GetConfigKeyByModel 通过模型名称获取配置 key
func (m *manager) GetConfigKeyByModel(modelName string) string {
	m.mux.RLock()
	defer m.mux.RUnlock()

	for key, conf := range m.configs {
		if _, ok := conf.GetModelIndexes()[modelName]; ok {
			return key
		}
	}

	return ""
}

func (m *manager) optimizeConfig() {
	for key, conf := range m.configs {
		usefulConnKey := make(map[string]struct{})
		// 遍历模型
		var usefulModels []config.DeviceModel
		for _, model := range conf.DeviceModels {
			if len(model.Devices) > 0 {
				usefulModels = append(usefulModels, model)
				// 遍历设备列表
				for _, device := range model.Devices {
					usefulConnKey[device.ConnectionKey] = struct{}{}
				}
			}
		}

		// 遍历连接
		usefulConnections := make(map[string]interface{})
		for k, conn := range conf.Connections {
			if _, ok := usefulConnKey[k]; ok {
				usefulConnections[k] = conn
			}
		}

		// 更新配置
		c := m.configs[key]
		c.DeviceModels = usefulModels
		c.Connections = usefulConnections
		c.ProtocolName = conf.ProtocolName
		c.Key = conf.Key
		m.configs[key] = c.UpdateIndexAndClean()
	}
}

// OptimizeConfig 优化所有驱动配置（移除未使用模型、移除未使用连接）
// 提示：此操作会修改并持久化驱动配置文件
func (m *manager) OptimizeConfig() {
	m.mux.Lock()
	defer m.mux.Unlock()

	m.optimizeConfig()
	// 保存
	for k, _ := range m.configs {
		_ = m.saveConfig(k)
	}
}

// GetModel 获取模型
func (m *manager) GetModel(modelName string) (config.DeviceModel, bool) {
	m.mux.RLock()
	defer m.mux.RUnlock()

	for _, conf := range m.configs {
		if index, ok := conf.GetModelIndexes()[modelName]; ok {
			return conf.DeviceModels[index], true
		}
	}
	return config.DeviceModel{}, false
}

// AddModel 新增模型
// 说明：仅用于新增模型，不用于更新模型
func (m *manager) AddModel(plugin string, model config.DeviceModel) error {
	if model.Name == "" {
		return errors.New("model name is null")
	}
	m.mux.Lock()
	defer m.mux.Unlock()

	// 插件是否存在
	for k, _ := range m.configs {
		if m.configs[k].ProtocolName == plugin {
			if _, ok := m.configs[k].GetModelIndexes()[model.Name]; ok {
				// 模型已存在
				return nil
			}
			// 新增模型
			newConfig := m.configs[k]
			newConfig.DeviceModels = append(newConfig.DeviceModels, model)
			newConfig = newConfig.UpdateIndexAndClean()
			m.configs[k] = newConfig
			// 持久化
			return m.saveConfig(k)
		}
	}

	// 插件不存在，创建新配置文件
	conf := m.createConfig(plugin)
	conf.DeviceModels = []config.DeviceModel{model}

	// 新增配置
	return m.addConfig(conf)
}

// AddOrUpdateDevice 新增或更新设备
func (m *manager) AddOrUpdateDevice(device config.Device) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	// 模型校验
	if device.ModelName == "" {
		return ErrDeviceModelNameNull
	}

	for s, conf := range m.configs {
		if modelIndex, ok := conf.GetModelIndexes()[device.ModelName]; ok {
			if deviceIndex, exist := conf.DeviceModels[modelIndex].GetDeviceIndexes()[device.ID]; exist {
				// 更新
				conf.DeviceModels[modelIndex].Devices[deviceIndex] = device
			} else {
				// 新增
				conf.DeviceModels[modelIndex].Devices = append(conf.DeviceModels[modelIndex].Devices, device)
			}
			m.configs[s] = conf.UpdateIndexAndClean()
			return m.saveConfig(s)
		}
	}

	return ErrUnknownModel
}

// GetDevice 获取设备
func (m *manager) GetDevice(modelName string, deviceID string) (config.Device, bool) {
	m.mux.RLock()
	defer m.mux.RUnlock()

	for _, conf := range m.configs {
		if modelIndex, ok := conf.GetModelIndexes()[modelName]; ok {
			if deviceIndex, exist := conf.DeviceModels[modelIndex].GetDeviceIndexes()[deviceID]; exist {
				device := conf.DeviceModels[modelIndex].Devices[deviceIndex]
				// 补充设备信息
				device.ModelName = modelName
				return device, true
			}
		}
	}
	return config.Device{}, false
}

// RemoveDevice 删除设备
func (m *manager) RemoveDevice(modelName string, deviceID string) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	for s, conf := range m.configs {
		if modelIndex, ok := conf.GetModelIndexes()[modelName]; ok {
			if deviceIndex, exist := conf.DeviceModels[modelIndex].GetDeviceIndexes()[deviceID]; exist {
				conf.DeviceModels[modelIndex].Devices = append(conf.DeviceModels[modelIndex].Devices[:deviceIndex], conf.DeviceModels[modelIndex].Devices[deviceIndex+1:]...)
				m.configs[s] = conf.UpdateIndexAndClean()
				return m.saveConfig(s)
			}
		}
	}
	return ErrUnknownDevice
}

// RemoveDeviceByID 根据 ID 删除设备
// 提示：性能消耗过高，推荐使用 RemoveDevice 方法进行删除
func (m *manager) RemoveDeviceByID(id string) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	for s, conf := range m.configs {
		// 遍历所有模型
		for i, model := range conf.DeviceModels {
			// 遍历所有设备
			for j, device := range model.Devices {
				if device.ID == id {
					m.configs[s].DeviceModels[i].Devices = append(model.Devices[:j], model.Devices[j+1:]...)
					m.configs[s] = conf.UpdateIndexAndClean()
					return m.saveConfig(s)
				}
			}
		}
	}

	return nil
}

// BatchRemoveDevice 批量删除设备
func (m *manager) BatchRemoveDevice(ids []string) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	// 便于检索
	idMap := make(map[string]struct{})
	for _, id := range ids {
		idMap[id] = struct{}{}
	}

	for s, conf := range m.configs {
		var changed bool
		for i, model := range conf.DeviceModels {
			if len(model.Devices) > 0 {
				newDevice := make([]config.Device, 0, len(model.Devices))
				for _, device := range model.Devices {
					if _, exist := idMap[device.ID]; exist {
						// 删除
						changed = true
					} else {
						// 保留
						newDevice = append(newDevice, device)
					}
				}
				// 更新设备列表
				if changed {
					m.configs[s].DeviceModels[i].Devices = newDevice
				}
			}
		}
		if changed {
			m.configs[s] = conf.UpdateIndexAndClean()
			return m.saveConfig(s)
		}
	}

	return nil
}

// AddConnection 新增连接
// 说明：仅用于新增连接，不用于更新连接
func (m *manager) AddConnection(plugin string, key string, conn any) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	// 插件是否存在
	for k, _ := range m.configs {
		if m.configs[k].ProtocolName == plugin {
			if _, ok := m.configs[k].Connections[key]; ok {
				// 连接已存在
				return nil
			}
			m.configs[k].Connections[key] = conn
			// 持久化
			return m.saveConfig(k)
		}
	}

	// 插件不存在，创建新配置文件
	conf := m.createConfig(plugin)
	conf.Connections[key] = conn

	// 新增配置
	return m.addConfig(conf)
}

// AddConfig 新增配置
func (m *manager) AddConfig(c config.Config) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	return m.addConfig(c)
}

// GetConnection 获取连接
func (m *manager) GetConnection(key string) (any, error) {
	m.mux.RLock()
	defer m.mux.RUnlock()

	for _, conf := range m.configs {
		if conn, ok := conf.Connections[key]; ok {
			return conn, nil
		}
	}

	return nil, nil
}

// RemoveConnection 删除连接
func (m *manager) RemoveConnection(key string) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	for s, conf := range m.configs {
		if _, ok := conf.Connections[key]; ok {
			delete(conf.Connections, key)
			m.configs[s] = conf.UpdateIndexAndClean()
			return m.saveConfig(s)
		}
	}

	return nil
}

// GetConnections 获取指定插件下所有连接配置
func (m *manager) GetConnections(plugin string) (map[string]interface{}, error) {
	m.mux.RLock()
	defer m.mux.RUnlock()

	for _, conf := range m.configs {
		if conf.ProtocolName == plugin {
			result := make(map[string]interface{}, len(conf.Connections))
			for k, v := range conf.Connections {
				result[k] = v
			}
			return result, nil
		}
	}

	return nil, ErrUnknownPlugin
}

// addConfig 新增配置
func (m *manager) addConfig(c config.Config) error {
	for s, conf := range m.configs {
		if conf.ProtocolName == c.ProtocolName {
			// 合并模型、设备
			for _, model := range c.DeviceModels {
				if modelIndex, ok := conf.GetModelIndexes()[model.Name]; ok {
					// 合并设备
					conf.DeviceModels[modelIndex].Devices = m.mergeDevices(conf.DeviceModels[modelIndex].Devices, model.Devices)
				} else {
					// 新增
					conf.DeviceModels = append(conf.DeviceModels, model)
				}
			}
			// 合并连接
			for k, _ := range c.Connections {
				conf.Connections[k] = c.Connections[k]
			}

			m.configs[s] = conf.UpdateIndexAndClean()
			return m.saveConfig(s)
		}
	}

	// 新增配置
	dirName := m.generateDirName(c.ProtocolName)
	// fix：填充配置文件 Key 字段
	c.Key = dirName
	m.configs[dirName] = c.UpdateIndexAndClean()
	return m.saveConfig(dirName)
}

// createDir 创建目录
func (m *manager) createDir(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(path, 0755); err != nil {
			return err
		}
	}
	return nil
}

// getSubDirs 获取子目录
func (m *manager) getSubDirs() []string {
	var dirs []string
	_ = filepath.WalkDir(m.root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			dirs = append(dirs, d.Name())
		}
		return nil
	})
	return dirs
}

// parseConfigFromFile 从文件解析配置
func (m *manager) parseConfigFromFile(path string) (config.Config, error) {
	if !m.fileExists(path) {
		return config.Config{}, ErrConfigNotExist
	}
	// 读取文件
	bytes, err := os.ReadFile(path)
	if err != nil {
		return config.Config{}, err
	}
	// 空文件校验
	if len(bytes) == 0 {
		return config.Config{}, ErrConfigEmpty
	}
	// json 解析
	var c config.Config
	if err = json.Unmarshal(bytes, &c); err != nil {
		return config.Config{}, err
	}
	return c, nil
}

// fileExists 判断文件是否存在
func (m *manager) fileExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

// generateDirName 生成目录名称
func (m *manager) generateDirName(protocolName string) string {
	return protocolName
}

// saveConfig 配置持久化到文件
func (m *manager) saveConfig(key string) error {
	// json 编码
	bytes, err := json.MarshalIndent(m.configs[key], "", "\t")
	if err != nil {
		return fmt.Errorf("config json encode error: %w", err)
	}
	// 创建目录
	if err = os.MkdirAll(filepath.Join(m.root, key), 0755); err != nil {
		return fmt.Errorf("config dir create error: %w", err)
	}
	// 写入文件
	fileName := filepath.Join(m.root, key, m.configName)
	if err := os.WriteFile(fileName, bytes, 0644); err != nil {
		return fmt.Errorf("config save to file error: %w", err)
	}

	return nil
}

// createConfig 创建配置
func (m *manager) createConfig(protocolName string) config.Config {
	return config.Config{
		DeviceModels: make([]config.DeviceModel, 0),
		Connections:  make(map[string]interface{}),
		ProtocolName: protocolName,
	}
}

// mergeDevice 合并设备
// 以 arr1 为主，arr2 为辅，合并 arr1 与 arr2 相同的设备，并返回合并后的设备列表
func (m *manager) mergeDevices(arr1 []config.Device, arr2 []config.Device) []config.Device {
	ids := make(map[string]struct{})
	var result []config.Device

	all := append(arr1, arr2...)
	for _, device := range all {
		if _, exist := ids[device.ID]; exist {
			continue
		}

		ids[device.ID] = struct{}{}
		result = append(result, device)
	}

	return result
}

// GetPluginNameByModel 获取模型所属的插件名称
func (m *manager) GetPluginNameByModel(modelName string) string {
	m.mux.RLock()
	defer m.mux.RUnlock()

	for _, conf := range m.configs {
		if _, ok := conf.GetModelIndexes()[modelName]; ok {
			return conf.ProtocolName
		}
	}

	return ""
}

// GetPluginNameByConnection 获取连接所属的插件名称
func (m *manager) GetPluginNameByConnection(key string) string {
	m.mux.RLock()
	defer m.mux.RUnlock()

	for _, conf := range m.configs {
		if _, ok := conf.Connections[key]; ok {
			return conf.ProtocolName
		}
	}

	return ""
}
