package main

import (
	"github.com/ibuilding-x/driver-box/driverbox"
	"github.com/ibuilding-x/driver-box/driverbox/export"
	"github.com/ibuilding-x/driver-box/driverbox/export/edgex"
	"os"
)

func main() {
	//localMode("127.0.0.1", "59999", "192.168.16.35")
	//_ = os.Setenv("EDGEX_SECURITY_SECRET_STORE", "false")

	driverbox.Start([]export.Export{&edgex.EdgexExport{}})
	select {}
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
