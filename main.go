package main

import (
	"driver-box/driver"
	"driver-box/driver/export"
	"driver-box/driver/export/edgex"
	"driver-box/driver/plugins"
	"driver-box/driver/plugins/httpserver"
	"os"
)

func main() {
	localMode("127.0.0.1", "59999", "192.168.16.35")
	_ = os.Setenv("EDGEX_SECURITY_SECRET_STORE", "false")

	plugins.Manager.Register("bacnet", &httpserver.Plugin{})
	driver.Start([]export.Export{edgex.NewEdgexExport()})
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
