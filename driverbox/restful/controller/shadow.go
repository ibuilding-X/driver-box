package controller

import (
	"encoding/json"
	"errors"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/helper/shadow"
	"github.com/ibuilding-x/driver-box/driverbox/restful/request"
	"io"
	"net/http"
)

var (
	snRequiredErr         = errors.New("sn is required")
	snPointRequiredErr    = errors.New("sn and point is required")
	unknownDevicePointErr = errors.New("unknown device point")
)

type Shadow struct {
}

// All 获取影子所有设备数据
func (s *Shadow) All(_ *http.Request) (any, error) {
	devices := helper.DeviceShadow.GetDevices()
	result := make([]shadow.DeviceAPI, 0)
	for _, device := range devices {
		result = append(result, device.ToDeviceAPI())
	}
	return result, nil
}

// Device 设备相关操作
func (s *Shadow) Device(r *http.Request) (any, error) {
	// 获取查询参数
	sn := r.URL.Query().Get("sn")

	switch r.Method {
	case http.MethodGet: // 查询
		if sn == "" {
			return nil, snRequiredErr
		}
		result, err := s.queryDevice(sn)
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
		err = s.updateDevice(req)
		if err != nil {
			return nil, err
		}
		return nil, nil
	default:
		return nil, errors.New(http.StatusText(http.StatusMethodNotAllowed))
	}
}

// DevicePoint 获取设备点位数据
func (s *Shadow) DevicePoint(r *http.Request) (any, error) {
	// 获取查询参数
	sn := r.URL.Query().Get("sn")
	point := r.URL.Query().Get("point")

	if sn == "" || point == "" {
		return nil, snPointRequiredErr
	}

	// 查询点位
	result, err := s.queryDevicePoint(sn, point)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// queryDevice 查询设备数据
func (s *Shadow) queryDevice(sn string) (any, error) {
	device, err := helper.DeviceShadow.GetDevice(sn)
	if err != nil {
		return nil, err
	}
	return device.ToDeviceAPI(), nil
}

// queryDevicePoint 查询指定点位数据
func (s *Shadow) queryDevicePoint(sn string, point string) (any, error) {
	device, err := helper.DeviceShadow.GetDevice(sn)
	if err != nil {
		return nil, err
	}
	if result, ok := device.GetDevicePointAPI(point); ok {
		return result, nil
	}
	return nil, unknownDevicePointErr
}

// updateDevice 更新设备影子数据
func (s *Shadow) updateDevice(data request.UpdateDeviceReq) error {
	for i, _ := range data {
		err := helper.DeviceShadow.SetDevicePoint(data[i].SN, data[i].Name, data[i].Value)
		if err != nil {
			return err
		}
	}
	return nil
}
