package controller

import (
	"encoding/json"
	"errors"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/internal/restful/request"
	"io"
	"net/http"
)

var (
	snRequiredErr         = errors.New("sn is required")
	unknownDevicePointErr = errors.New("unknown device point")
)

type Shadow struct {
	*commonController
}

func NewShadow() *Shadow {
	return &Shadow{
		commonController: &commonController{},
	}
}

// All 获取影子所有设备数据
func (s *Shadow) All(w http.ResponseWriter, _ *http.Request) {
	list := helper.DeviceShadow.All()
	s.commonController.Success(w, list)
}

// Device 设备相关操作
func (s *Shadow) Device(w http.ResponseWriter, r *http.Request) {
	// 获取查询参数
	sn := r.URL.Query().Get("sn")
	point := r.URL.Query().Get("point")

	switch r.Method {
	case http.MethodGet: // 查询
		if sn == "" {
			s.commonController.Error(w, http.StatusBadRequest, snRequiredErr, nil)
			return
		}
		var result any
		var err error
		if point == "" {
			// 查询设备
			result, err = s.queryDevice(sn)
		}
		// 查询点位
		result, err = s.queryDevicePoint(sn, point)
		if err != nil {
			s.commonController.Error(w, http.StatusBadRequest, err, nil)
			return
		}
		s.commonController.Success(w, result)
	case http.MethodPost: // 更新
		// 读取 body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			s.commonController.Error(w, http.StatusBadRequest, err, nil)
			return
		}
		defer r.Body.Close()
		// 解析 body
		var req request.UpdateDeviceReq
		err = json.Unmarshal(body, &req)
		if err != nil {
			s.commonController.Error(w, http.StatusBadRequest, err, nil)
			return
		}
		err = s.updateDevice(req)
		if err != nil {
			s.commonController.Error(w, http.StatusInternalServerError, err, nil)
			return
		}
		s.commonController.Success(w, nil)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
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
