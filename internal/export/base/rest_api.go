package base

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sort"
	"time"

	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/driverbox/shadow"
	"github.com/ibuilding-x/driver-box/internal/cache"
	"github.com/ibuilding-x/driver-box/internal/core"
	"github.com/ibuilding-x/driver-box/internal/export/base/restful"
	"github.com/ibuilding-x/driver-box/internal/export/base/restful/request"
	"github.com/ibuilding-x/driver-box/internal/export/base/restful/route"
	"github.com/ibuilding-x/driver-box/internal/logger"
	shadow0 "github.com/ibuilding-x/driver-box/internal/shadow"
	"github.com/ibuilding-x/driver-box/pkg/config"
	"github.com/ibuilding-x/driver-box/pkg/library"
	"go.uber.org/zap"
)

func (export *Export) registerApi() {
	restful.HandleFunc(http.MethodGet, route.V1Prefix+"ok", func(h *http.Request) (any, error) {
		return true, nil
	})

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

	//资源库服务
	restful.HandleFunc(http.MethodGet, route.V1Prefix+"library/model/get", libraryModelGet)

	//sse服务
	//http.HandleFunc("/sse/log", func(w http.ResponseWriter, r *http.Request) {
	//	include := r.URL.Query().Get("include")
	//	exclude := r.URL.Query().Get("exclude")
	//	w.Header().Set("Content-Type", "text/event-stream")
	//	w.Header().Set("Cache-Control", "no-cache")
	//	w.Header().Set("Connection", "keep-alive")
	//	w.Header().Set("Access-Control-Allow-Origin", "*")
	//
	//	//定义一个channel，注册至logger
	//	loggerChannel := make(chan []byte, 100)
	//	logger.ChanWriter.Add(loggerChannel)
	//	defer func() {
	//		logger.ChanWriter.Remove(loggerChannel)
	//		close(loggerChannel)
	//		logger.Logger.Info("sse client disconnected")
	//	}()
	//	for bytes := range loggerChannel {
	//		// 将消息格式化为SSE格式
	//		message := string(bytes)
	//		if len(include) > 0 && strings.Index(message, include) == -1 {
	//			continue
	//		}
	//		if len(exclude) > 0 && strings.Index(message, exclude) != -1 {
	//			continue
	//		}
	//		// 写入响应体
	//		_, e := w.Write(bytes)
	//		if e != nil {
	//			break
	//		}
	//		//刷新
	//		w.(http.Flusher).Flush()
	//	}
	//})
	// 第五步：启动 REST 服务
	go func() {
		srv = &http.Server{Addr: ":" + export.httpListen, Handler: restful.HttpRouter}
		e := srv.ListenAndServe()
		if e != nil {
			logger.Logger.Error("start rest server error", zap.Error(e))
		}
	}()
}

type kv map[string]interface{}

var (
	snRequiredErr         = errors.New("sn is required")
	snPointRequiredErr    = errors.New("sn and point is required")
	unknownDevicePointErr = errors.New("unknown device point")
)

// All 获取影子所有设备数据
func getAllDevices(_ *http.Request) (any, error) {
	devices := shadow0.Shadow().GetDevices()
	//按DeviceID排序
	sort.Slice(devices, func(i, j int) bool {
		return devices[i].ID < devices[j].ID
	})

	//定义个结构体，改变UpdatedAt的格式
	type Point struct {
		shadow.DevicePoint
		UpdatedAt string `json:"updatedAt"`
		WriteAt   string `json:"writeAt"`
	}
	type Device struct {
		shadow.Device
		Points    map[string]Point `json:"points"`
		UpdatedAt string           `json:"updatedAt"`
	}
	list := make([]Device, len(devices))
	for i, device := range devices {
		//获取设备所有点位
		points := make(map[string]Point)
		for k, v := range device.Points {
			points[k] = Point{
				DevicePoint: v,
				UpdatedAt:   v.UpdatedAt.Format(time.DateTime),
				WriteAt:     v.WriteAt.Format(time.DateTime),
			}
		}

		list[i] = Device{
			Device:    device,
			Points:    points,
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
	device, ok := shadow0.Shadow().GetDevice(sn)
	if !ok {
		return nil, errors.New("unknown device")
	}
	return device, nil
}

// queryDevicePoint 查询指定点位数据
func queryDevicePoint(sn string, point string) (any, error) {
	p, err := shadow0.Shadow().GetDevicePointDetails(sn, point)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// updateDevice 更新设备影子数据
func updateDevice(data request.UpdateDeviceReq) error {
	for i, _ := range data {
		err := shadow0.Shadow().SetDevicePoint(data[i].ID, data[i].Name, data[i].Value)
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
	return shadow0.Shadow().GetDevicePoint(sn, point)
}

// 获取设备列表
func deviceList(r *http.Request) (any, error) {
	type Device struct {
		config.Device
		Points []config.Point `json:"points"`
	}
	devices := make([]Device, 0)
	for _, device := range cache.Get().Devices() {
		points, _ := cache.Get().GetPoints(device.ModelName)
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
	device, ok := cache.Get().GetDevice(sn)
	if !ok {
		return nil, errors.New("device not found")
	}
	return device, nil
}

func libraryModelGet(r *http.Request) (any, error) {
	key := r.URL.Query().Get("key")
	return library.Model().LoadLibrary(key)
}
