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
)

type Manager interface {
	SetConfigPath(path string)
	SetConfigFileName(name string)
	SetScriptFileName(name string)

	LoadConfig() error
	GetConfigs() map[string]config.Config

	GetModel(modelName string) (config.DeviceModel, bool)
	GetDevice(modelName string, deviceName string) (config.Device, bool)
	AddOrUpdateDevice(device config.Device) error
	RemoveDevice(modelName string, deviceName string) error

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
			if deviceIndex, exist := conf.DeviceModels[modelIndex].GetDeviceIndexes()[device.DeviceSn]; exist {
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
func (m *manager) GetDevice(modelName string, deviceName string) (config.Device, bool) {
	m.mux.RLock()
	defer m.mux.RUnlock()

	for _, conf := range m.configs {
		if modelIndex, ok := conf.GetModelIndexes()[modelName]; ok {
			if deviceIndex, exist := conf.DeviceModels[modelIndex].GetDeviceIndexes()[deviceName]; exist {
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
func (m *manager) RemoveDevice(modelName string, deviceName string) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	for s, conf := range m.configs {
		if modelIndex, ok := conf.GetModelIndexes()[modelName]; ok {
			if deviceIndex, exist := conf.DeviceModels[modelIndex].GetDeviceIndexes()[deviceName]; exist {
				conf.DeviceModels[modelIndex].Devices = append(conf.DeviceModels[modelIndex].Devices[:deviceIndex], conf.DeviceModels[modelIndex].Devices[deviceIndex+1:]...)
				m.configs[s] = conf.UpdateIndexAndClean()
				return m.saveConfig(s)
			}
		}
	}
	return ErrUnknownDevice
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
	// 写入文件
	fileName := filepath.Join(m.root, key, m.configName)
	if err := os.WriteFile(fileName, bytes, 0644); err != nil {
		return fmt.Errorf("config save to file error: %w", err)
	}

	return nil
}
