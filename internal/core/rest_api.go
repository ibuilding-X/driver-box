package core

import (
	"encoding/json"
	"errors"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/driverbox/restful"
	"github.com/ibuilding-x/driver-box/driverbox/restful/route"
	"io"
	"net/http"
)

// 注册restapi
func RegisterApi() {
	//设备API
	restful.HandleFunc(http.MethodPost, route.DevicePointWrite, writePoint)
	restful.HandleFunc(http.MethodPost, route.DevicePointsWrite, writePoints)
	restful.HandleFunc(http.MethodGet, route.DevicePointRead, readPoint)
	restful.HandleFunc(http.MethodGet, route.DeviceList, deviceList)
	restful.HandleFunc(http.MethodGet, route.DeviceGet, deviceGet)
}

// 写入某个设备点位
func writePoint(r *http.Request) (any, error) {
	sn := r.URL.Query().Get("id")
	point := r.URL.Query().Get("point")
	value := r.URL.Query().Get("value")
	return nil, SendSinglePoint(sn, plugin.WriteMode, plugin.PointData{
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
	return nil, SendBatchWrite(data.ID, data.Values)
}

// 读取某个设备点位
func readPoint(r *http.Request) (any, error) {
	sn := r.URL.Query().Get("id")
	point := r.URL.Query().Get("point")
	e := SendSinglePoint(sn, plugin.ReadMode, plugin.PointData{
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
