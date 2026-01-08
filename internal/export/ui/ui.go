package ui

import (
	"net/http"
	"sort"
	"strings"
	"text/template"

	"github.com/ibuilding-x/driver-box/pkg/driverbox/config"
	"github.com/ibuilding-x/driver-box/pkg/driverbox/helper"
	"github.com/ibuilding-x/driver-box/pkg/driverbox/helper/utils"
	"github.com/julienschmidt/httprouter"
	"go.uber.org/zap"
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
	config.Point
	Value     interface{}
	Update    string
	Writeable bool
}
type DeviceDetail struct {
	//Points []Point
}

func deviceDetail(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	//设备信息
	device, _ := helper.CoreCache.GetDevice(params.ByName("deviceId"))

	//点位信息
	shadowDevice, _ := helper.DeviceShadow.GetDevice(params.ByName("deviceId"))
	points := make([]Point, 0)
	modelPoints, _ := helper.CoreCache.GetPoints(device.ModelName)
	for _, point := range modelPoints {
		p, _ := helper.DeviceShadow.GetDevicePointDetails(shadowDevice.ID, point.Name())
		points = append(points, Point{
			Point:     point,
			Value:     p.Value,
			Update:    p.UpdatedAt.Format("2006-01-02 15:04:05"),
			Writeable: point.ReadWrite() != config.ReadWrite_R,
		})
	}
	//points按name排序
	sort.Slice(points, func(i, j int) bool {
		return points[i].Name() < points[j].Name()
	})

	data := struct {
		Device Device
		Points []Point
	}{
		Device: Device{
			ID:   device.ID,
			Name: device.Description,
		},
		Points: points,
	}

	vendor("./res/ui/device_detail.tmpl", writer, data)
}

func vendor(tmpl string, writer http.ResponseWriter, data interface{}) {
	index := strings.LastIndex(tmpl, "/")
	t, err := template.New(tmpl[index+1:]).Funcs(template.FuncMap{
		"contains": func(s interface{}, substr string) bool {
			return strings.Contains(s.(string), substr)
		},
		"eq": func(a, b interface{}) bool {
			s1, _ := utils.Conv2String(a)
			s2, _ := utils.Conv2String(b)
			return s1 == s2
		},
	}).ParseFiles(tmpl)
	if err != nil {
		return
	}
	e := t.Execute(writer, data)
	if e != nil {
		helper.Logger.Error("vendor", zap.Error(e))
	}
}
