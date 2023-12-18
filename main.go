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
	var exportTopic string

	flag.StringVar(&broker, "broker", os.Getenv("MQTT_BROKER"), "mqttExport: broker address")
	flag.StringVar(&clientId, "clientId", os.Getenv("MQTT_CLIENT_ID"), "mqttExport: clientId")
	flag.StringVar(&username, "username", os.Getenv("MQTT_USERNAME"), "mqttExport: username")
	flag.StringVar(&password, "password", os.Getenv("MQTT_PASSWORD"), "mqttExport: password")
	flag.StringVar(&exportTopic, "exportTopic", os.Getenv("MQTT_EXPORT_TOPIC"), "mqttExport: exportTopic")
	flag.Parse()

	if len(clientId) == 0 {
		clientId = ""
	}
	helper.DriverConfig.DefaultDeviceTTL = 5
	driverbox.Start([]export.Export{&export.MqttExport{
		Broker:      broker,
		ClientID:    clientId,
		Username:    username,
		Password:    password,
		ExportTopic: exportTopic,
	}})
	select {}
}

func localMode() {
	_ = os.Setenv("MQTT_BROKER", "mqtt://127.0.0.1:1883")
	_ = os.Setenv("MQTT_CLIENT_ID", "123456")
}
