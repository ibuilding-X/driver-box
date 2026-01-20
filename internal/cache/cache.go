package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/internal/export"
	"github.com/ibuilding-x/driver-box/internal/logger"
	"github.com/ibuilding-x/driver-box/internal/shadow"
	"github.com/ibuilding-x/driver-box/pkg/config"
	"github.com/ibuilding-x/driver-box/pkg/crontab"
	"github.com/ibuilding-x/driver-box/pkg/event"
	"github.com/ibuilding-x/driver-box/pkg/fileutil"
	"go.uber.org/zap"
)

var (
	// ErrConfigNotExist is the error returned when the config file does not exist.
	ErrConfigNotExist = errors.New("config not exist")
	// ErrConfigEmpty is the error returned when the config file is empty.
	ErrConfigEmpty = errors.New("config is empty")
)

// coreCache 核心缓存
type CoreCache interface {
	GetModel(modelName string) (model config.DeviceModel, ok bool) // model info
	// 查询指定模型的所有点，并保持该点位在配置文件中的有序性
	GetPoints(modelName string) ([]config.Point, bool)
	GetDevice(id string) (device config.Device, ok bool)
	GetPointByModel(modelName string, pointName string) (point config.Point, ok bool) // search point by model
	GetPointByDevice(id string, pointName string) (point config.Point, ok bool)       // search point by device
	Models() (models []config.DeviceModel)                                            // all model
	Devices() (devices []config.Device)

	UpdateDeviceProperty(id string, key string, value string) error // 更新设备属性
	DeleteDevice(id string)                                         // 删除设备
	UpdateDeviceDesc(id string, desc string)                        // 更新设备描述
	Reset()
	// AddConnection 新增连接
	AddConnection(plugin string, key string, conn any) error
	// GetConnection 获取连接信息
	GetConnection(key string) any
	// AddModel 新增模型
	AddModel(plugin string, model config.DeviceModel) error
	// AddOrUpdateDevice 新增或更新设备
	AddOrUpdateDevice(device config.Device) error
	// BatchRemoveDevice 批量删除设备
	BatchRemoveDevice(ids []string) error

	//将指定插件的配置进行持久化刷新
	Flush(pluginName string)
	// 将所有插件的配置进行持久化刷新
	FlushAll()
}

var instance *cache
var once = &sync.Once{}

type cache struct {
	//各协议插件的配置缓存
	configs map[string]configCache
	//设备缓存
	devices map[string]*config.Device
	//模型缓存
	models map[string]*config.DeviceModel
	mutex  *sync.RWMutex // 锁
}

func Get() CoreCache {
	once.Do(func() {
		instance = &cache{
			devices: make(map[string]*config.Device),
			configs: make(map[string]configCache),
			models:  make(map[string]*config.DeviceModel),
			mutex:   &sync.RWMutex{},
		}
	})
	return instance
}

// InitCoreCache 初始化核心缓存
func InitCoreCache(plugins map[string]plugin.Plugin) (obj CoreCache, err error) {
	obj = Get()
	for key, p := range plugins {
		instance.configs[key] = configCache{
			plugin: p,
		}
	}
	err = instance.loadConfig()
	if err != nil {
		return nil, err
	}

	configMap := instance.configs
	for key, _ := range configMap {
		for _, deviceModel := range configMap[key].Models {

			//modelName防重校验
			if preModel, ok := instance.models[deviceModel.Name]; ok {
				if preModel.Name != deviceModel.Name ||
					preModel.ModelID != deviceModel.ModelID {
					return instance, fmt.Errorf("conflict model base information: %v  %v",
						deviceModel.ModelBase, preModel.ModelBase)
				}
			} else {
				instance.models[deviceModel.Name] = &deviceModel
			}
			//点表基础校验
			for _, devicePoint := range deviceModel.DevicePoints {
				checkPoint(&deviceModel, &devicePoint)
			}
			for _, device := range deviceModel.Devices {
				if device.ID == "" {
					logger.Logger.Error("config error , device id is empty", zap.Any("device", device))
					continue
				}
				deviceId := device.ID
				device.ModelName = deviceModel.Name
				device.Protocol = key
				if deviceRaw, ok := instance.devices[deviceId]; !ok {
					instance.devices[deviceId] = &device
				} else {
					if deviceRaw.ModelName != device.ModelName {
						return instance, fmt.Errorf("conflict model for device [%s]: %s -> %s", device.ID,
							device.ModelName, deviceRaw.ModelName)
					}
				}
			}
		}
	}

	_, err = crontab.Instance().AddFunc("5s", func() {
		var configs map[string]configCache
		instance.mutex.RLock()
		configs = instance.configs
		instance.mutex.RUnlock()
		for protocol, cfg := range configs {
			if cfg.cacheModifyTime.After(cfg.fileModifyTime) && cfg.cacheModifyTime.Before(time.Now().Add(-5*time.Second)) {
				instance.Flush(protocol)
			}
		}
	})

	return instance, err
}

func GetConfig(pluginName string) config.Config {
	cfg, ok := instance.configs[pluginName]
	if !ok {
		return config.Config{}
	}
	models := make([]config.DeviceModel, 0, len(cfg.Models))
	for _, model := range cfg.Models {
		models = append(models, model)
	}
	return config.Config{
		Connections:  cfg.Connections,
		DeviceModels: models,
		ProtocolName: pluginName,
	}
}
func (c *cache) loadConfig() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	driverPath := path.Join(config.ResourcePath, "driver")
	// 自动创建配置目录
	if err := createDir(driverPath); err != nil {
		return err
	}

	// 遍历配置目录，获取所有文件夹
	dirs := getSubDirs(driverPath)
	if len(dirs) <= 0 {
		return nil
	}

	//config 协议唯一性
	// 解析每个文件夹的配置
	for i, _ := range dirs {
		path := filepath.Join(driverPath, dirs[i], "config.json")
		cfg, err := parseConfigFromFile(path)
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
		cfg.Key = dirs[i]

		curTime := time.Now()
		item := configCache{
			FilePath:        path,
			plugin:          c.configs[cfg.ProtocolName].plugin,
			Models:          make(map[string]config.DeviceModel),
			Connections:     make(map[string]interface{}),
			fileModifyTime:  curTime,
			cacheModifyTime: curTime,
		}
		for _, model := range cfg.DeviceModels {
			item.Models[model.Name] = model
		}
		if cfg.Connections != nil {
			item.Connections = cfg.Connections
		}
		c.configs[cfg.ProtocolName] = item
	}
	return nil
}
func parseConfigFromFile(path string) (config.Config, error) {
	if !fileutil.FileExists(path) {
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
	return c, c.Validate()
}

// createDir 创建目录
func createDir(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(path, 0755); err != nil {
			return err
		}
	}
	return nil
}
func getSubDirs(driverPath string) []string {
	var dirs []string
	_ = filepath.WalkDir(driverPath, func(path string, d fs.DirEntry, err error) error {
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

// 检查点位配置合法性
func checkPoint(model *config.DeviceModel, point *config.Point) {
	if point.Name() == "" {
		logger.Logger.Error("config error , point name is empty", zap.Any("point", point), zap.String("modelName", model.Name))
	}
	if point.Description() == "" {
		logger.Logger.Warn("config error , point description is empty", zap.Any("point", point), zap.String("model", model.Name))
	}
	valueType := point.ValueType()
	if valueType != config.ValueType_Float && valueType != config.ValueType_Int && valueType != config.ValueType_String {
		logger.Logger.Error("point valueType config error , valid config is: int float string", zap.Any("point", point), zap.String("model", model.Name))
	}
	reportModel := point.ReportMode()
	readWrite := point.ReadWrite()
	if readWrite != config.ReadWrite_RW && readWrite != config.ReadWrite_R && readWrite != config.ReadWrite_W {
		logger.Logger.Error("point readWrite config error , valid config is: R W RW", zap.Any("point", point), zap.String("model", model.Name))
	}
	if reportModel != config.ReportMode_Real && reportModel != config.ReportMode_Change {
		logger.Logger.Error("point reportMode config error , valid config is: realTime change period", zap.Any("point", point), zap.String("model", model.Name))
	}
	//存在精度换算时，点位类型要求float
	if point.Scale() != 0 && valueType != config.ValueType_Float {
		logger.Logger.Error("point scale config error , valid config is: float", zap.Any("point", point), zap.String("model", model.Name))
	}
}
func (c *cache) GetModel(modelName string) (model config.DeviceModel, ok bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	for _, conf := range c.configs {
		for _, m := range conf.Models {
			if m.Name == modelName {
				return m, true
			}
		}
	}
	return config.DeviceModel{}, false
}

func (c *cache) GetPoints(modelName string) ([]config.Point, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if model, exist := c.GetModel(modelName); exist {
		return model.DevicePoints, true
	}
	return make([]config.Point, 0), false
}
func (c *cache) GetDevice(id string) (config.Device, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if dev, exist := c.devices[id]; exist {
		return *dev, true
	}
	return config.Device{}, false
}

func (c *cache) GetPointByModel(modelName string, pointName string) (point config.Point, ok bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	model, ok := c.models[modelName]
	if !ok {
		return config.Point{}, false
	}
	//添加校验防止程序存在bug
	if model.Name != modelName {
		logger.Logger.Error("model name not match", zap.String("modelName", modelName), zap.String("model", model.Name))
		return config.Point{}, false
	}
	pointBase, ok := model.GetPoint(pointName)
	if !ok {
		return config.Point{}, false
	}
	//添加校验防止程序存在bug
	if pointBase.Name() != pointName {
		logger.Logger.Error("point name not match", zap.String("pointName", pointName), zap.Any("point", pointBase))
		return config.Point{}, false
	}
	return pointBase, true
}

// GetPointByDevice 查询指定设备的指定点位信息
func (c *cache) GetPointByDevice(id string, pointName string) (point config.Point, ok bool) {
	// 查询设备
	if device, ok := c.GetDevice(id); ok {
		return c.GetPointByModel(device.ModelName, pointName)
	}
	return config.Point{}, false
}

func GetRunningPluginByDevice(id string) (plugin plugin.Plugin, ok bool) {
	instance.mutex.RLock()
	defer instance.mutex.RUnlock()
	if key, ok := instance.devices[id]; ok {
		if cfg, ok := instance.configs[key.Protocol]; ok {
			return cfg.plugin, true
		}
		return nil, false
	}
	logger.Logger.Error("device not found plugin", zap.String("id", id))
	return nil, false
}

func (c *cache) Models() (models []config.DeviceModel) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	var results []config.DeviceModel
	for _, conf := range c.configs {
		for _, model := range conf.Models {
			results = append(results, model)
		}

	}
	return results
}

func (c *cache) Devices() (devices []config.Device) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	for _, dev := range c.devices {
		devices = append(devices, *dev)
	}
	return
}

func (c *cache) GetAllRunningPluginKey() []string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	keys := make([]string, 0)
	for key, _ := range c.configs {
		keys = append(keys, key)
	}
	return keys
}

// UpdateDeviceProperty 更新设备属性并持久化
func (c *cache) UpdateDeviceProperty(id string, key string, value string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if dev, ok := c.devices[id]; ok {
		if dev.Properties == nil {
			dev.Properties = make(map[string]string)
		}
		dev.Properties[key] = value
		// 持久化
		return c.AddOrUpdateDevice(*dev)
	}
	return fmt.Errorf("device %s not found", id)
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
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if dev, ok := c.devices[id]; ok {
		dev.Description = desc
		// 持久化
		_ = c.AddOrUpdateDevice(*dev)
	}
}

// Reset 重置数据
func (c *cache) Reset() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.configs = make(map[string]configCache)
	c.devices = make(map[string]*config.Device)
	c.models = make(map[string]*config.DeviceModel)
}

// AddOrUpdateDevice 添加或更新设备
// 更新内容列表
// * 核心缓存设备
// * 设备影子
// * 持久化文件
func (c *cache) AddOrUpdateDevice(dev config.Device) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if logger.Logger != nil {
		logger.Logger.Info("core cache add device", zap.Any("device", dev), zap.Any("model", dev.ModelName))
	}
	// 查找模型信息
	_, ok := c.GetModel(dev.ModelName)
	if !ok {
		logger.Logger.Error("model not found", zap.String("modelName", dev.ModelName))
		return fmt.Errorf("model %s not found", dev.ModelName)
	}
	// 校验设备是否已存在
	storedDeviceBase, ok := c.devices[dev.ID]
	if ok {
		if storedDeviceBase.ModelName != dev.ModelName {
			logger.Logger.Error("conflict model for device", zap.String("deviceId", dev.ID))
			return fmt.Errorf("conflict model for device [%s]: %s -> %s", dev.ID,
				dev.ModelName, storedDeviceBase.ModelName)
		}
	}

	// 自动补全设备描述
	if dev.Description == "" {
		dev.Description = dev.ID
	}

	if !ok {
		defer export.TriggerEvents(event.EventCodeAddDevice, dev.ID, nil)
	}
	c.devices[dev.ID] = &dev
	// 更新设备影子
	if !shadow.Shadow().HasDevice(dev.ID) {
		shadow.Shadow().AddDevice(dev.ID, dev.ModelName)
	}
	cfg := c.configs[dev.Protocol]
	cfg.cacheModifyTime = time.Now()
	c.configs[dev.Protocol] = cfg
	return nil
}

// AddConnection 新增连接
func (c *cache) AddConnection(plugin string, key string, conn any) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	cfg := c.configs[plugin]
	cfg.Connections[key] = conn
	cfg.cacheModifyTime = time.Now()
	c.configs[plugin] = cfg
	return nil
}

// GetConnection 获取连接信息
func (c *cache) GetConnection(key string) any {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	for _, conf := range c.configs {
		if conn, ok := conf.Connections[key]; ok {
			return conn
		}
	}
	return nil
}

// AddModel 新增模型
func (c *cache) AddModel(plugin string, model config.DeviceModel) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	cfg := c.configs[plugin]
	cfg.Models[model.Name] = model
	cfg.cacheModifyTime = time.Now()
	c.configs[plugin] = cfg
	return nil
}

// BatchRemoveDevice 批量删除设备
func (c *cache) BatchRemoveDevice(ids []string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	nowUnix := time.Now()
	for _, id := range ids {
		if dev, ok := c.devices[id]; ok {
			cfg := c.configs[dev.Protocol]
			cfg.cacheModifyTime = nowUnix
			c.configs[dev.Protocol] = cfg
		}
		export.TriggerEvents(event.EventCodeWillDeleteDevice, id, nil)
		delete(c.devices, id)
	}
	// 删除设备影子
	_ = shadow.Shadow().DeleteDevice(ids...)
	return nil
}

func (c *cache) Flush(pluginName string) {
	c.mutex.RLock()
	cfgCache, ok := c.configs[pluginName]
	if !ok {
		c.mutex.RUnlock()
		return
	}
	models := make([]config.DeviceModel, 0, len(cfgCache.Models))
	for _, model := range cfgCache.Models {
		models = append(models, model)
	}
	config := config.Config{
		Connections:  cfgCache.Connections,
		DeviceModels: models,
		ProtocolName: pluginName,
	}
	c.mutex.RUnlock()
	bytes, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		logger.Logger.Error("marshal config error", zap.Error(err))
		return
	}
	err = os.WriteFile(cfgCache.FilePath, bytes, 0644)
	if err != nil {
		logger.Logger.Error("write config file error", zap.Error(err))
		return
	}
	// 更新文件修改时间
	c.mutex.Lock()
	cfg := c.configs[pluginName]
	cfg.fileModifyTime = time.Now()
	c.configs[pluginName] = cfg
	c.mutex.Unlock()
}
func (c *cache) FlushAll() {
	c.mutex.RLock()
	keys := make([]string, 0, len(c.configs))
	for k := range c.configs {
		keys = append(keys, k)
	}
	c.mutex.RUnlock()
	for _, k := range keys {
		c.Flush(k)
	}
}
