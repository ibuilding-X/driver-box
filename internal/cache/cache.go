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

const configFile = "config.json"

var (
	// ErrConfigNotExist is the error returned when the config file does not exist.
	ErrConfigNotExist = errors.New("config not exist")
	// ErrConfigEmpty is the error returned when the config file is empty.
	ErrConfigEmpty = errors.New("config is empty")
)

// coreCache 核心缓存
type CoreCache interface {
	GetModel(modelName string) (model config.Model, ok bool) // model info
	// 查询指定模型的所有点，并保持该点位在配置文件中的有序性
	GetPoints(modelName string) ([]config.Point, bool)
	GetDevice(id string) (device config.Device, ok bool)
	GetPointByModel(modelName string, pointName string) (point config.Point, ok bool) // search point by model
	GetPointByDevice(id string, pointName string) (point config.Point, ok bool)       // search point by device
	Models() (models []config.Model)                                                  // all model
	Devices() (devices []config.Device)

	UpdateDeviceProperty(id string, key string, value string) error // 更新设备属性
	DeleteDevice(id string)                                         // 删除设备
	UpdateDeviceDesc(id string, desc string) error                  // 更新设备描述
	// AddConnection 新增连接
	AddConnection(plugin string, key string, conn any) error
	// GetConnection 获取连接信息
	GetConnection(key string) any
	// AddModel 新增模型
	AddModel(plugin string, model config.Model) error
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
	plugins map[string]cachePlugin
	//设备缓存
	devices map[string]cacheDevice
	//模型缓存
	models map[string]cacheModel
	//连接换成
	connections map[string]cacheConnection
	mutex       *sync.RWMutex // 锁
}

func Get() CoreCache {
	once.Do(func() {
		instance = &cache{
			plugins:     make(map[string]cachePlugin),
			devices:     make(map[string]cacheDevice),
			models:      make(map[string]cacheModel),
			connections: make(map[string]cacheConnection),
			mutex:       &sync.RWMutex{},
		}
	})
	return instance
}

// InitCoreCache 初始化核心缓存
func InitCoreCache(plugins map[string]plugin.Plugin) (obj CoreCache, err error) {
	obj = Get()
	for key, p := range plugins {
		instance.plugins[key] = cachePlugin{
			plugin: p,
		}
	}
	err = instance.loadConfig(plugins)
	if err != nil {
		return nil, err
	}

	_, err = crontab.Instance().AddFunc("5s", func() {
		var configs map[string]cachePlugin
		instance.mutex.RLock()
		configs = instance.plugins
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
	instance.mutex.RLock()
	defer instance.mutex.RUnlock()
	return convertConfig(pluginName)
}
func convertConfig(pluginName string) config.Config {
	_, ok := instance.plugins[pluginName]
	if !ok {
		return config.Config{}
	}
	models := make([]config.DeviceModel, 0, len(instance.models))
	for _, model := range instance.models {
		devices := make([]config.Device, 0)
		for _, device := range instance.devices {
			if device.ModelName == model.Name {
				devices = append(devices, device.Device)
			}
		}
		models = append(models, config.DeviceModel{
			Model:   model.Model,
			Devices: devices,
		})
	}
	connections := make(map[string]interface{})
	for key, connection := range instance.connections {
		if connection.pluginName == pluginName {
			connections[key] = connection.connection
		}
	}
	return config.Config{
		Connections:  connections,
		DeviceModels: models,
		PluginName:   pluginName,
	}
}
func (c *cache) loadConfig(plugins map[string]plugin.Plugin) error {
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

	curTime := time.Now()
	//config 协议唯一性
	// 解析每个文件夹的配置
	for i, _ := range dirs {
		path := filepath.Join(driverPath, dirs[i], configFile)
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

		p, ok := plugins[cfg.PluginName]
		if !ok {
			return errors.New("plugin " + cfg.PluginName + " not found")
		}
		//相同插件不允许存在多个文件中
		pl, ok := c.plugins[cfg.PluginName]
		if !ok {
			return errors.New("plugin " + cfg.PluginName + " unSupport!")
		}
		if pl.plugin != p {
			return errors.New("invalid plugin " + cfg.PluginName + "!")
		}
		if len(pl.FilePath) > 0 && pl.FilePath != path {
			return errors.New("plugin " + cfg.PluginName + " already exists in " + pl.FilePath)
		}

		//构建coreCache的缓存结构
		pl.FilePath = path
		pl.fileModifyTime = curTime
		pl.cacheModifyTime = curTime
		c.plugins[cfg.PluginName] = pl

		for _, model := range cfg.DeviceModels {
			for _, device := range model.Devices {
				if device.ID == "" {
					logger.Logger.Error("config error , device id is empty", zap.Any("device", device))
					continue
				}
				_, ok = c.devices[device.ID]
				if ok {
					logger.Logger.Error("device exists！", zap.Any("device", device))
					continue
				}
				device.ModelName = model.Name
				device.PluginName = cfg.PluginName
				c.devices[device.ID] = cacheDevice{
					device,
				}
			}
			points := make(map[string]*config.Point)
			for k, _ := range model.DevicePoints {
				point := &model.DevicePoints[k]
				checkPoint(&model.Model, point)
				points[point.Name()] = point
			}
			//释放内存
			model.Devices = nil
			c.models[model.Name] = cacheModel{
				Model:      model.Model,
				pluginName: cfg.PluginName,
				points:     points,
			}

		}
		for key, connection := range cfg.Connections {
			c.connections[key] = cacheConnection{
				connection: connection,
				pluginName: cfg.PluginName,
			}
		}
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
	return c, nil
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
func checkPoint(model *config.Model, point *config.Point) {
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
func (c *cache) GetModel(modelName string) (config.Model, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	m, ok := c.models[modelName]
	if ok {
		return m.Model, true
	}
	return config.Model{}, false
}

func (c *cache) GetPoints(modelName string) ([]config.Point, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	m, ok := c.models[modelName]
	if !ok {
		return make([]config.Point, 0), false
	}
	return m.DevicePoints, true
}
func (c *cache) GetDevice(id string) (config.Device, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if dev, exist := c.devices[id]; exist {
		return dev.Device, true
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

	pointBase, ok := model.points[pointName]
	if !ok {
		return config.Point{}, false
	}
	//添加校验防止程序存在bug
	if pointBase.Name() != pointName {
		logger.Logger.Error("point name not match", zap.String("pointName", pointName), zap.Any("point", pointBase))
		return config.Point{}, false
	}
	return *pointBase, true
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
		if cfg, ok := instance.plugins[key.PluginName]; ok {
			return cfg.plugin, true
		}
		return nil, false
	}
	logger.Logger.Error("device not found plugin", zap.String("id", id))
	return nil, false
}

func (c *cache) Models() (models []config.Model) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	var results []config.Model
	for _, model := range c.models {
		results = append(results, model.Model)

	}
	return results
}

func (c *cache) Devices() (devices []config.Device) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	for _, dev := range c.devices {
		devices = append(devices, dev.Device)
	}
	return
}

func (c *cache) GetAllRunningPluginKey() []string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	keys := make([]string, 0)
	for key, _ := range c.plugins {
		keys = append(keys, key)
	}
	return keys
}

// UpdateDeviceProperty 更新设备属性并持久化
func (c *cache) UpdateDeviceProperty(id string, key string, value string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	dev, ok := c.devices[id]
	if !ok {
		return errors.New("device " + id + " not found")
	}
	if dev.Properties == nil {
		dev.Properties = make(map[string]string)
	}
	dev.Properties[key] = value
	cfg := c.plugins[dev.PluginName]
	cfg.cacheModifyTime = time.Now()
	c.plugins[dev.PluginName] = cfg
	return nil
}

// DeleteDevice 删除设备
func (c *cache) DeleteDevice(id string) {
	e := c.BatchRemoveDevice([]string{id})
	if e != nil {
		logger.Logger.Error("remove device error", zap.String("id", id))
	}
}

// UpdateDeviceDesc 更新设备描述
func (c *cache) UpdateDeviceDesc(id string, desc string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	dev, ok := c.devices[id]
	if !ok {
		return errors.New("device " + id + " not found")
	}
	dev.Description = desc
	cfg := c.plugins[dev.PluginName]
	cfg.cacheModifyTime = time.Now()
	return nil
}

// Reset 重置数据
func Reset() {
	instance.mutex.Lock()
	defer instance.mutex.Unlock()
	instance.plugins = make(map[string]cachePlugin)
	instance.devices = make(map[string]cacheDevice)
	instance.models = make(map[string]cacheModel)
	instance.connections = make(map[string]cacheConnection)
}

// AddOrUpdateDevice 添加或更新设备
// 更新内容列表
// * 核心缓存设备
// * 设备影子
// * 持久化文件
func (c *cache) AddOrUpdateDevice(dev config.Device) error {
	if len(dev.ModelName) == 0 {
		return errors.New("device modelName is empty")
	}
	if dev.ID == "" {
		return errors.New("device id is empty")
	}
	// 自动补全设备描述
	if dev.Description == "" {
		dev.Description = dev.ID
	}
	c.mutex.Lock()
	defer c.mutex.Unlock()
	logger.Logger.Info("core cache add device", zap.Any("device", dev))

	//未匹配到模型
	model, ok := c.models[dev.ModelName]
	if !ok {
		logger.Logger.Error("model not found", zap.String("modelName", dev.ModelName))
		return fmt.Errorf("model %s not found", dev.ModelName)
	}

	dev.PluginName = model.pluginName

	storedDeviceBase, ok := c.devices[dev.ID]
	// 校验设备是否已存在
	if ok {
		if storedDeviceBase.ModelName != dev.ModelName {
			logger.Logger.Error("conflict model for device", zap.String("deviceId", dev.ID))
			return fmt.Errorf("conflict model for device [%s]: %s -> %s", dev.ID,
				dev.ModelName, storedDeviceBase.ModelName)
		}
	} else {
		defer export.TriggerEvents(event.EventCodeAddDevice, dev.ID, nil)
	}
	c.devices[dev.ID] = cacheDevice{
		Device: dev,
	}

	// 更新设备影子
	if !shadow.Shadow().HasDevice(dev.ID) {
		shadow.Shadow().AddDevice(dev.ID, dev.ModelName)
	}
	p := c.plugins[model.pluginName]
	p.cacheModifyTime = time.Now()
	c.plugins[dev.PluginName] = p
	return nil
}

// AddConnection 新增连接
func (c *cache) AddConnection(plugin string, key string, conn any) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	cfg, ok := c.plugins[plugin]
	if !ok {
		return errors.New("plugin " + plugin + " not exists")
	}
	_, ok = c.connections[key]
	if ok {
		return errors.New("connection " + key + " already exists")
	}
	c.connections[key] = cacheConnection{
		connection: conn,
		pluginName: plugin,
	}
	cfg.cacheModifyTime = time.Now()
	c.plugins[plugin] = cfg
	return nil
}

// GetConnection 获取连接信息
func (c *cache) GetConnection(key string) any {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if conn, ok := c.connections[key]; ok {
		return conn.connection
	}
	return nil
}

// AddModel 新增模型
func (c *cache) AddModel(pluginName string, model config.Model) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	p, exists := c.plugins[pluginName]
	if !exists {
		return errors.New("plugin " + pluginName + " not exists")
	}
	_, ok := c.models[model.Name]
	if ok {
		return errors.New("model " + model.Name + " already exists")
	}
	points := make(map[string]*config.Point)
	for i, point := range model.DevicePoints {
		points[point.Name()] = &model.DevicePoints[i]
	}
	//释放模型点位内存空间
	model.DevicePoints = nil
	c.models[model.Name] = cacheModel{
		Model:  model,
		points: points,
	}
	p.cacheModifyTime = time.Now()
	c.plugins[pluginName] = p
	return nil
}

// BatchRemoveDevice 批量删除设备
func (c *cache) BatchRemoveDevice(ids []string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	plugins := make(map[string]string)
	for _, id := range ids {
		if dev, ok := c.devices[id]; ok {
			plugins[dev.PluginName] = dev.PluginName
		}
		export.TriggerEvents(event.EventCodeWillDeleteDevice, id, nil)
		delete(c.devices, id)
	}
	nowUnix := time.Now()
	for _, p := range plugins {
		cfg := c.plugins[p]
		cfg.cacheModifyTime = nowUnix
		c.plugins[p] = cfg
	}
	// 删除设备影子
	_ = shadow.Shadow().DeleteDevice(ids...)
	return nil
}

func (c *cache) Flush(pluginName string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	p := c.plugins[pluginName]
	cfg := convertConfig(pluginName)

	//闲置插件
	if len(cfg.Connections) == 0 && len(cfg.Connections) == 0 {
		//无接入配置，需删除
		if len(p.FilePath) > 0 {
			err := os.Remove(p.FilePath)
			if err != nil {
				logger.Logger.Error("remove file error", zap.String("file", p.FilePath), zap.Error(err))
			}

		}
		return
	}

	if p.FilePath == "" {
		p.FilePath = path.Join(config.ResourcePath, "driver", pluginName, configFile)
	}
	bytes, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		logger.Logger.Error("marshal config error", zap.Error(err))
		return
	}
	err = os.WriteFile(p.FilePath, bytes, 0644)
	if err != nil {
		logger.Logger.Error("write config file error", zap.Error(err))
		return
	}
	p.fileModifyTime = time.Now()
	c.plugins[pluginName] = p
}
func (c *cache) FlushAll() {
	c.mutex.RLock()
	keys := make([]string, 0, len(c.plugins))
	for k := range c.plugins {
		keys = append(keys, k)
	}
	c.mutex.RUnlock()
	for _, k := range keys {
		c.Flush(k)
	}
}
