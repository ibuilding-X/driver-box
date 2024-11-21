package ui

import (
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"text/template"
)

type Device struct {
	ID         string
	Name       string
	ModeId     string
	Plugin     string
	Status     string
	Connection string
}

func devices(writer http.ResponseWriter, request *http.Request, _ httprouter.Params) {
	data := struct {
		Devices []Device
	}{
		Devices: make([]Device, 0),
	}
	for _, device := range helper.CoreCache.Devices() {

		dev := Device{
			ID:         device.ID,
			Name:       device.Description,
			Connection: device.ConnectionKey,
		}
		//在离线状态
		shadow, ok := helper.DeviceShadow.GetDevice(device.ID)
		if ok && shadow.Online {
			dev.Status = "online"
		} else {
			dev.Status = "offline"
		}
		//关联模型
		model, ok := helper.CoreCache.GetModel(device.ModelName)
		if ok {
			dev.ModeId = model.ModelID
		}
		//关联插件
		dev.Plugin = helper.CoreCache.GetConnectionPluginName(device.ConnectionKey)
		data.Devices = append(data.Devices, dev)
	}
	t, err := template.ParseFiles("./res/ui/devices.tmpl")
	if err != nil {
		return
	}
	t.Execute(writer, data)
}

type Point struct {
	Name        string
	Description string
	Value       interface{}
	Update      string
}
type DeviceDetail struct {
	//Points []Point
}

func deviceDetail(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	shadowDevice, _ := helper.DeviceShadow.GetDevice(params.ByName("deviceId"))

	points := make([]Point, 0)
	for _, point := range shadowDevice.Points {
		p, _ := helper.CoreCache.GetPointByDevice(shadowDevice.ID, point.Name)
		points = append(points, Point{
			Name:        point.Name,
			Description: p.Description,
			Value:       point.Value,
			Update:      point.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	data := struct {
		Device DeviceDetail
		Points []Point
	}{
		Device: DeviceDetail{},
		Points: points,
	}

	t, err := template.ParseFiles("./res/ui/device_detail.tmpl")
	if err != nil {
		return
	}
	t.Execute(writer, data)
}
