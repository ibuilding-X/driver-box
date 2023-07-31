package main

import (
	"flag"
	"github.com/ibuilding-x/driver-box/driverbox"
	"github.com/ibuilding-x/driver-box/driverbox/export"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"os"
)

func main() {
	//localMode()
	var broker string
	var clientId string
	var username string
	var password string

	flag.StringVar(&broker, "broker", os.Getenv("MQTT_BROKER"), "mqttExport: broker address")
	flag.StringVar(&clientId, "clientId", os.Getenv("MQTT_CLIENT_ID"), "mqttExport: clientId")
	flag.StringVar(&clientId, "username", os.Getenv("MQTT_USERNAME"), "mqttExport: username")
	flag.StringVar(&clientId, "password", os.Getenv("MQTT_PASSWORD"), "mqttExport: password")
	flag.Parse()

	if len(clientId) == 0 {
		clientId = ""
	}
	helper.DriverConfig.DefaultDeviceTTL = 900
	driverbox.Start([]export.Export{&export.MqttExport{
		Broker:   broker,
		ClientID: clientId,
		Username: username,
		Password: password,
	}})
}

func localMode() {
	_ = os.Setenv("MQTT_BROKER", "mqtt://127.0.0.1:1883")
	_ = os.Setenv("MQTT_CLIENT_ID", "123456")
}
