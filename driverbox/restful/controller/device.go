package controller

import (
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/internal/core"
	"net/http"
)

type Device struct {
}

// 写入某个设备点位
func (s *Device) WritePoint(r *http.Request) (any, error) {
	sn := r.URL.Query().Get("sn")
	point := r.URL.Query().Get("name")
	value := r.URL.Query().Get("value")
	return nil, core.SendSinglePoint(sn, plugin.WriteMode, plugin.PointData{
		PointName: point,
		Value:     value,
	})
}

// 读取某个设备点位
func (s *Device) ReadPoint(r *http.Request) (any, error) {
	sn := r.URL.Query().Get("sn")
	point := r.URL.Query().Get("name")
	e := core.SendSinglePoint(sn, plugin.ReadMode, plugin.PointData{
		PointName: point,
	})
	if e != nil {
		return nil, e
	}
	return helper.DeviceShadow.GetDevicePoint(sn, point)
}
