package main

import (
	"github.com/ibuilding-x/driver-box/driverbox"
	"github.com/ibuilding-x/driver-box/driverbox/export"
	"os"
)

func main() {
	broker := os.Getenv("MQTT_BROKER")
	clientId := os.Getenv("MQTT_CLIENT_ID")
	username := os.Getenv("MQTT_USERNAME")
	password := os.Getenv("MQTT_PASSWORD")
	driverbox.Start([]export.Export{&export.MqttExport{
		Broker:   broker,
		ClientID: clientId,
		Username: username,
		Password: password,
	}})
}
