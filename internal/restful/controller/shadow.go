package controller

import (
	"encoding/json"
	"errors"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/internal/restful/request"
	"github.com/julienschmidt/httprouter"
	"io"
	"net/http"
)

var (
	unknownDeviceErr      = errors.New("unknown device")
	unknownDevicePointErr = errors.New("unknown device point")
	setDevicePointErr     = errors.New("set device point error")
)

type h map[string]any

type Shadow struct {
	*commonController
}

type pointValue struct {
	Point string `json:"point"`
	Value any    `json:"value"`
}

func NewShadow() *Shadow {
	return &Shadow{
		commonController: &commonController{},
	}
}

// All 获取影子所有设备数据
func (s *Shadow) All(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	list := helper.DeviceShadow.All()
	s.commonController.Success(writer, list)
}

// QueryDevice 查询影子指定设备所有信息
func (s *Shadow) QueryDevice(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	sn := params.ByName("sn")
	points, err := helper.DeviceShadow.GetDevicePoints(sn)
	if err != nil {
		s.commonController.Error(writer, http.StatusBadRequest, unknownDeviceErr, nil)
		return
	}
	s.commonController.Success(writer, points)
}

// QueryDevicePoint 查询指定设备点位信息
func (s *Shadow) QueryDevicePoint(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	sn := params.ByName("sn")
	point := params.ByName("point")
	v, err := helper.DeviceShadow.GetDevicePoint(sn, point)
	if err != nil {
		s.commonController.Error(writer, http.StatusBadRequest, unknownDevicePointErr, nil)
		return
	}
	s.commonController.Success(writer, h{
		"value": v,
	})
}

// UpdateDevicePoints 更新设备点位信息
func (s *Shadow) UpdateDevicePoints(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	sn := params.ByName("sn")
	// 读取 body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.commonController.Error(w, http.StatusInternalServerError, err, nil)
		return
	}
	defer r.Body.Close()
	// 解析 body
	var req request.UpdateDevicePointsReq
	err = json.Unmarshal(body, &req)
	if err != nil {
		s.commonController.Error(w, http.StatusBadRequest, err, nil)
		return
	}
	// 设置
	for i, _ := range req {
		err = helper.DeviceShadow.SetDevicePoint(sn, req[i].Point, req[i].Value)
		if err != nil {
			s.commonController.Error(w, http.StatusInternalServerError, err, nil)
			return
		}
	}
	s.commonController.Success(w, nil)
}
