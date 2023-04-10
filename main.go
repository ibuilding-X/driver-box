package main

import (
	"driver-box/driver"
	"github.com/edgexfoundry/device-sdk-go/v2/pkg/startup"
	"os"
)

const (
	serviceName string = "driver-box"
	version     string = "0.0.2"
)

func main() {
	_ = os.Setenv("EDGEX_SECURITY_SECRET_STORE", "false")

	// 正式环境需注释掉
	//serverHost := "127.0.0.1"
	//serverPort := "59999"
	//EdgeXServerHost := "192.168.122.66"
	//_ = os.Setenv("SERVICE_HOST", serverHost)
	//_ = os.Setenv("SERVICE_PORT", serverPort)
	//_ = os.Setenv("REGISTRY_HOST", EdgeXServerHost)
	//_ = os.Setenv("CLIENTS_CORE_DATA_HOST", EdgeXServerHost)
	//_ = os.Setenv("CLIENTS_CORE_METADATA_HOST", EdgeXServerHost)
	//_ = os.Setenv("MESSAGEQUEUE_HOST", EdgeXServerHost)

	sd := driver.Driver{}
	startup.Bootstrap(serviceName, version, &sd)
}
