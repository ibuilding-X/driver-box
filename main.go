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
	//localMode("192.168.16.88", "59999", "127.0.0.1")

	sd := driver.Driver{}
	startup.Bootstrap(serviceName, version, &sd)
}

// localMode 本地调试模式
// serverHost：驱动服务监听地址
// serverPort：驱动服务监听端口号
// edgeXServerHost：EdgeX服务地址
// 示例： localMode("127.0.0.1", "59999", "192.168.16.88")
func localMode(serverHost, serverPort, edgeXServerHost string) {
	_ = os.Setenv("SERVICE_HOST", serverHost)
	_ = os.Setenv("SERVICE_PORT", serverPort)
	_ = os.Setenv("REGISTRY_HOST", edgeXServerHost)
	_ = os.Setenv("CLIENTS_CORE_DATA_HOST", edgeXServerHost)
	_ = os.Setenv("CLIENTS_CORE_METADATA_HOST", edgeXServerHost)
	_ = os.Setenv("MESSAGEQUEUE_HOST", edgeXServerHost)
}
