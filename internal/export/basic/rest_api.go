package basic

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/common"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/helper/cmanager"
	"github.com/ibuilding-x/driver-box/driverbox/library"
	"github.com/ibuilding-x/driver-box/driverbox/models"
	"github.com/ibuilding-x/driver-box/driverbox/pkg/shadow"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/driverbox/restful"
	"github.com/ibuilding-x/driver-box/driverbox/restful/request"
	"github.com/ibuilding-x/driver-box/driverbox/restful/route"
	"github.com/ibuilding-x/driver-box/internal/bootstrap"
	"github.com/ibuilding-x/driver-box/internal/core"
	"github.com/ibuilding-x/driver-box/internal/logger"
	"go.uber.org/zap"
	"io"
	"net/http"
	"os"
	"path"
	"sort"
	"strings"
	"time"
)

func registerApi() {
	// 插件 REST API
	restful.HandleFunc(http.MethodGet, route.V1Prefix+"plugin/cache/get", getCache)
	restful.HandleFunc(http.MethodPost, route.V1Prefix+"plugin/cache/set", setCache)
	// 核心配置 API
	restful.HandleFunc(http.MethodPost, route.V1Prefix+"config/update", updateCoreConfig)

	// 设备影子 API
	restful.HandleFunc(http.MethodGet, route.V1Prefix+"shadow/all", getAllDevices)
	restful.HandleFunc(http.MethodGet, route.V1Prefix+"shadow/device", deviceShadow)
	restful.HandleFunc(http.MethodGet, route.V1Prefix+"shadow/devicePoint", getDevicePoint)

	//设备API
	restful.HandleFunc(http.MethodPost, route.DevicePointWrite, writePoint)
	restful.HandleFunc(http.MethodPost, route.DevicePointsWrite, writePoints)
	restful.HandleFunc(http.MethodGet, route.DevicePointRead, readPoint)
	restful.HandleFunc(http.MethodGet, route.DeviceList, deviceList)
	restful.HandleFunc(http.MethodGet, route.DeviceGet, deviceGet)
	restful.HandleFunc(http.MethodPost, route.DeviceAdd, deviceAdd)

	//资源库服务
	restful.HandleFunc(http.MethodGet, route.V1Prefix+"library/model/get", libraryModelGet)

	//sse服务
	http.HandleFunc("/sse/log", func(w http.ResponseWriter, r *http.Request) {
		include := r.URL.Query().Get("include")
		exclude := r.URL.Query().Get("exclude")
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		//定义一个channel，注册至logger
		loggerChannel := make(chan []byte, 100)
		logger.ChanWriter.Add(loggerChannel)
		defer func() {
			logger.ChanWriter.Remove(loggerChannel)
			close(loggerChannel)
			logger.Logger.Info("sse client disconnected")
		}()
		for bytes := range loggerChannel {
			// 将消息格式化为SSE格式
			message := string(bytes)
			if len(include) > 0 && strings.Index(message, include) == -1 {
				continue
			}
			if len(exclude) > 0 && strings.Index(message, exclude) != -1 {
				continue
			}
			// 写入响应体
			_, e := w.Write(bytes)
			if e != nil {
				break
			}
			//刷新
			w.(http.Flusher).Flush()
		}
	})
}

type kv map[string]interface{}

// Get 获取信息
// 返回数据结构：{"key":"value"}
func getCache(r *http.Request) (any, error) {
	// 获取查询 Key
	key := r.URL.Query().Get("key")
	if key == "" {
		return nil, errors.New("key cannot be empty")
	}

	// 响应
	value, ok := helper.PluginCacheMap.Load(key)
	if !ok {
		value = ""
	}
	obj := kv{key: value}
	return obj, nil
}

// Set 存储信息
// body 示例：{"key", "value"}
func setCache(r *http.Request) (any, error) {
	// 读取 body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("read body error: %s", err)
	}
	defer r.Body.Close()
	// 键值对解析
	var obj kv
	if err = json.Unmarshal(body, &body); err != nil {
		return nil, fmt.Errorf("json decode error: %s", err)
	}
	// 存储
	for key, value := range obj {
		helper.PluginCacheMap.Store(key, value)
	}
	// 响应
	return nil, nil
}

func updateCoreConfig(r *http.Request) (any, error) {
	// ------------------------------------------------------------
	// 配置文件覆盖更新
	// ------------------------------------------------------------
	// 接收核心配置
	body, err := io.ReadAll(r.Body)
	if err != nil {
		helper.Logger.Error("read body error", zap.Error(err))
		return nil, fmt.Errorf("read body error: %s", err)
	}
	defer r.Body.Close()

	// 数据解析
	var list []models.APIConfig
	err = json.Unmarshal(body, &list)
	if err != nil {
		helper.Logger.Error("config json decode error", zap.Error(err))
		return nil, fmt.Errorf("config json decode error: %s", err)
	}
	if len(list) == 0 {
		helper.Logger.Error("request body is empty")
		return nil, errors.New("request body is empty")
	}

	// 保存核心配置
	for _, config := range list {
		dir := path.Join(helper.EnvConfig.ConfigPath, config.Key)
		// 删除旧配置
		_ = os.RemoveAll(dir)
		// 创建文件夹
		_ = os.Mkdir(dir, 0755)
		// 保存 config.json / converter.lua 文件
		configFileName := path.Join(dir, common.CoreConfigName)
		scriptFilename := path.Join(dir, common.LuaScriptName)
		configData, _ := json.MarshalIndent(config.Config, "", "\t")

		err = os.WriteFile(configFileName, configData, 0666)
		if err != nil {
			helper.Logger.Error("save config.json file error", zap.Error(err))
		}
		if config.Config.ProtocolName == "modbus" && config.Script == "" {
			continue
		}
		err = os.WriteFile(scriptFilename, []byte(config.Script), 0666)
		if err != nil {
			helper.Logger.Error("save converter.lua file error", zap.Error(err))
		}
	}

	// ------------------------------------------------------------
	// plugins 重载
	// ------------------------------------------------------------
	// 1. 停止所有 timerTask 任务
	helper.Crontab.Stop()

	// 2. 停止运行中的 plugin
	pluginKeys := helper.CoreCache.GetAllRunningPluginKey()
	if len(pluginKeys) > 0 {
		for i, _ := range pluginKeys {
			if plugin, ok := helper.CoreCache.GetRunningPluginByKey(pluginKeys[i]); ok {
				err = plugin.Destroy()
				if err != nil {
					helper.Logger.Error("stop plugin error", zap.String("plugin", pluginKeys[i]), zap.Error(err))
				} else {
					helper.Logger.Info("stop plugin success", zap.String("plugin", pluginKeys[i]))
				}
			}
		}
	}
	// 3. 停止影子服务设备状态监听、删除影子服务
	helper.DeviceShadow.StopStatusListener()

	// 4. 加载 plugins
	err = bootstrap.LoadPlugins()
	if err != nil {
		return nil, fmt.Errorf("load plugins error: %s", err)
	}

	return nil, nil
}

var (
	snRequiredErr         = errors.New("sn is required")
	snPointRequiredErr    = errors.New("sn and point is required")
	unknownDevicePointErr = errors.New("unknown device point")
)

// All 获取影子所有设备数据
func getAllDevices(_ *http.Request) (any, error) {
	devices := helper.DeviceShadow.GetDevices()
	//按DeviceID排序
	sort.Slice(devices, func(i, j int) bool {
		return devices[i].ID < devices[j].ID
	})

	//定义个结构体，改变UpdatedAt的格式
	type Device struct {
		shadow.Device
		UpdatedAt string `json:"updatedAt"`
	}
	list := make([]Device, len(devices))
	for i, device := range devices {
		list[i] = Device{
			Device:    device,
			UpdatedAt: device.UpdatedAt.Format(time.DateTime),
		}
	}
	return list, nil
}

// Device 设备相关操作
func deviceShadow(r *http.Request) (any, error) {
	// 获取查询参数
	sn := r.URL.Query().Get("id")

	switch r.Method {
	case http.MethodGet: // 查询
		if sn == "" {
			return nil, snRequiredErr
		}
		result, err := queryDevice(sn)
		if err != nil {
			return nil, err
		}
		return result, nil
	case http.MethodPost: // 更新
		// 读取 body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		defer r.Body.Close()
		// 解析 body
		var req request.UpdateDeviceReq
		err = json.Unmarshal(body, &req)
		if err != nil {
			return nil, err
		}
		err = updateDevice(req)
		if err != nil {
			return nil, err
		}
		return nil, nil
	default:
		return nil, errors.New(http.StatusText(http.StatusMethodNotAllowed))
	}
}

// DevicePoint 获取设备点位数据
func getDevicePoint(r *http.Request) (any, error) {
	// 获取查询参数
	sn := r.URL.Query().Get("id")
	point := r.URL.Query().Get("point")

	if sn == "" || point == "" {
		return nil, snPointRequiredErr
	}

	// 查询点位
	result, err := queryDevicePoint(sn, point)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// queryDevice 查询设备数据
func queryDevice(sn string) (any, error) {
	device, ok := helper.DeviceShadow.GetDevice(sn)
	if !ok {
		return nil, errors.New("unknown device")
	}
	return device, nil
}

// queryDevicePoint 查询指定点位数据
func queryDevicePoint(sn string, point string) (any, error) {
	p, err := helper.DeviceShadow.GetDevicePointDetails(sn, point)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// updateDevice 更新设备影子数据
func updateDevice(data request.UpdateDeviceReq) error {
	for i, _ := range data {
		err := helper.DeviceShadow.SetDevicePoint(data[i].ID, data[i].Name, data[i].Value)
		if err != nil {
			return err
		}
	}
	return nil
}

// 写入某个设备点位
func writePoint(r *http.Request) (any, error) {
	sn := r.URL.Query().Get("id")
	point := r.URL.Query().Get("point")
	value := r.URL.Query().Get("value")
	return nil, core.SendSinglePoint(sn, plugin.WriteMode, plugin.PointData{
		PointName: point,
		Value:     value,
	})
}

// 批量写入某个设备的多个点位
// curl -X POST -H "Content-Type: application/json" -d '{"id":"deviceId","values":[{"name":"pointName","value":"1"}]}' http://127.0.0.1:8081/api/v1/device/writePoints
func writePoints(r *http.Request) (any, error) {
	if r.Method != http.MethodPost {
		return nil, errors.New(http.StatusText(http.StatusMethodNotAllowed))
	}
	// 读取 body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	// 解析 body
	var data plugin.DeviceData
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}
	return nil, core.SendBatchWrite(data.ID, data.Values)
}

// 读取某个设备点位
func readPoint(r *http.Request) (any, error) {
	sn := r.URL.Query().Get("id")
	point := r.URL.Query().Get("point")
	e := core.SendSinglePoint(sn, plugin.ReadMode, plugin.PointData{
		PointName: point,
	})
	if e != nil {
		return nil, e
	}
	return helper.DeviceShadow.GetDevicePoint(sn, point)
}

// 获取设备列表
func deviceList(r *http.Request) (any, error) {
	type Device struct {
		config.Device
		Points []config.Point `json:"points"`
	}
	devices := make([]Device, 0)
	for _, device := range helper.CoreCache.Devices() {
		points, _ := helper.CoreCache.GetPoints(device.ModelName)
		devices = append(devices, Device{
			Device: device,
			Points: points,
		})
	}
	return devices, nil
}

// 获取设备信息
func deviceGet(r *http.Request) (any, error) {
	sn := r.URL.Query().Get("id")
	device, ok := helper.CoreCache.GetDevice(sn)
	if !ok {
		return nil, errors.New("device not found")
	}
	return device, nil
}

// 获取设备信息
func deviceAdd(r *http.Request) (any, error) {
	//读取body中的json内容
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return false, err
	}
	defer r.Body.Close()
	//解析body
	var cfg config.Config
	err = json.Unmarshal(body, &cfg)
	if err != nil {
		return false, err
	}
	err = cmanager.AddConfig(cfg)
	if err != nil {
		return false, err
	}
	return true, nil
}

func libraryModelGet(r *http.Request) (any, error) {
	key := r.URL.Query().Get("key")
	return library.Model().LoadLibrary(key)
}
