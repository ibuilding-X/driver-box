package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/dto"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

// 配置文件名称
const configFileName string = "config.json"

type Config struct {
	// 持久化目录
	path string
	// 配置列表
	configs map[string]*dto.Config
	// 读写锁
	mu sync.RWMutex
}

func New(path string) *Config {
	return &Config{
		path: path,
	}
}

// Load 加载配置
func (c *Config) Load() error {
	if c.path == "" {
		return nil
	}

	// 创建目录
	if err := os.MkdirAll(c.path, 0755); err != nil {
		return err
	}

	// 遍历目录
	var dirs []string
	err := filepath.WalkDir(c.path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			dirs = append(dirs, d.Name())
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(dirs) == 0 {
		return nil
	}
	for _, dir := range dirs {
		filePath := filepath.Join(c.path, dir, configFileName)
		config, err := readFile(filePath)
		if err != nil {
			return err
		}

		// 保存配置
		c.mu.Lock()
		c.configs[dir] = &config
		c.mu.Unlock()
	}

	return nil
}

// Add 新增配置
func (c *Config) Add(config dto.Config) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if config.ProtocolName == "" {
		return errors.New("config.protocolName can not be empty")
	}

	// 自动补全
	config = complete(config)

	for i, _ := range c.configs {
		if c.configs[i].ProtocolName == config.ProtocolName {
			targetConfig := c.configs[i]
			// 合并模型
			for modelName, model := range config.Models {
				if _, ok := targetConfig.Models[modelName]; ok {
					// 合并点位
					for pointName, point := range model.Points {
						targetConfig.Models[modelName].Points[pointName] = point
					}
					// 合并设备
					for deviceId, device := range model.Devices {
						targetConfig.Models[modelName].Devices[deviceId] = device
					}
				} else {
					// 追加
					targetConfig.Models[modelName] = model
				}
			}

			// 合并连接
			for connKey, connection := range config.Connections {
				targetConfig.Connections[connKey] = connection
			}

			return nil
		}
	}

	// 新增
	config = optimize(config)
	c.configs[config.ProtocolName] = &config
	return c.save(config.ProtocolName)
}

// Get 获取配置
func (c *Config) Get(protocolName string) (dto.Config, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for i, _ := range c.configs {
		if c.configs[i].ProtocolName == protocolName {
			return clone(*c.configs[i]), nil
		}
	}

	return dto.Config{}, errors.New("config not found")
}

func (c *Config) AddModel(model dto.Model) error {
	if model.Name == "" {
		return errors.New("model name can not be empty")
	}
	if model.ProtocolName == "" {
		return errors.New("model.protocolName can not be empty")
	}

	return nil
}

func (c *Config) AddDevice(device dto.Device) error {

	return nil
}

func (c *Config) GetModelByName(name string) (dto.Model, error) {

	return dto.Model{}, nil
}

func (c *Config) GetDeviceById(id string) (dto.Device, error) {

	return dto.Device{}, nil
}

func (c *Config) DeleteDeviceById(id string) error {

	return nil
}

func (c *Config) BatchDeleteDevices(ids []string) error {

	return nil
}

func (c *Config) save(key string) error {
	config := c.configs[key]
	metadata := configToMetadata(*config)
	// json 编码
	bytes, err := json.MarshalIndent(metadata, "", "\t")
	if err != nil {
		return fmt.Errorf("config json encode error: %w", err)
	}
	// 创建目录
	if err = os.MkdirAll(filepath.Join(c.path, key), 0755); err != nil {
		return fmt.Errorf("config dir create error: %w", err)
	}
	// 写入文件
	fileName := filepath.Join(c.path, key, configFileName)
	if err := os.WriteFile(fileName, bytes, 0644); err != nil {
		return fmt.Errorf("config save to file error: %w", err)
	}
	return nil
}
