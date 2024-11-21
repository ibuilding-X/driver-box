package ui

import (
	"github.com/ibuilding-x/driver-box/driverbox/helper"
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

func devices(writer http.ResponseWriter, request *http.Request) {
	t, err := template.ParseFiles("./res/ui/devices.tmpl")
	if err != nil {
		return
	}
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
	t.Execute(writer, data)
}
