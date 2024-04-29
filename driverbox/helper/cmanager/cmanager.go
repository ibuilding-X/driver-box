package cmanager

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

const (
	// DefaultConfigPath is the default path to the config directory.
	DefaultConfigPath string = "./drivers"
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
)

var instance = New()

type Manager interface {
	SetConfigPath(path string)
	SetConfigFileName(name string)
	SetScriptFileName(name string)

	LoadConfig() error
	GetConfigs() map[string]config.Config

	GetModel(modelName string) (config.DeviceModel, bool)
	GetDevice(modelName string, deviceID string) (config.Device, bool)
	AddOrUpdateDevice(device config.Device) error
	RemoveDevice(modelName string, deviceID string) error
	RemoveDeviceByID(id string) error

	BatchRemoveDevice(ids []string) error

	AddConfig(c config.Config) error
}

type manager struct {
	root       string
	configName string
	scriptName string
	configs    map[string]config.Config
	mux        *sync.RWMutex
}

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

func GetModel(modelName string) (config.DeviceModel, bool) {
	return instance.GetModel(modelName)
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

func AddConfig(c config.Config) error {
	return instance.AddConfig(c)
}

func BatchRemoveDevice(ids []string) error {
	return instance.BatchRemoveDevice(ids)
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

	// 自动创建配置目录
	if err := m.createDir(m.root); err != nil {
		return err
	}

	// 遍历配置目录，获取所有文件夹
	dirs := m.getSubDirs()
	if len(dirs) <= 0 {
		return nil
	}

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
			return err
		}
		// 保存配置
		m.configs[dirs[i]] = c.UpdateIndexAndClean()
	}

	return nil
}

// GetConfigs 获取所有配置
func (m *manager) GetConfigs() map[string]config.Config {
	m.mux.RLock()
	defer m.mux.RUnlock()

	return m.configs
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

// AddConfig 新增配置
func (m *manager) AddConfig(c config.Config) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	for s, conf := range m.configs {
		if conf.ProtocolName == c.ProtocolName {
			// 合并模型、设备
			for _, model := range c.DeviceModels {
				if modelIndex, ok := conf.GetModelIndexes()[model.Name]; ok {
					// 更新
					conf.DeviceModels[modelIndex] = model
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
	return fmt.Sprintf("%s-%s", protocolName, strconv.FormatInt(time.Now().UnixMilli(), 36))
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
