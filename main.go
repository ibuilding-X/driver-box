package main

import (
	"driver-box/core/helper"
	"driver-box/driver/bootstrap"
	"driver-box/driver/export"
	"driver-box/driver/export/edgex"
	"os"
)

func main() {
	localMode("127.0.0.1", "59999", "192.168.16.35")
	_ = os.Setenv("EDGEX_SECURITY_SECRET_STORE", "false")

	//第一步：加载配置文件DriverConfig
	helper.Export = &export.DefaultExport{}

	//第二步：初始化日志记录器
	if err := helper.InitLogger("DEBUG"); err != nil {
		return
	}

	//第三步：启动driver-box插件
	if err := bootstrap.LoadPlugins(); err != nil {
		helper.Logger.Error(err.Error())
		return
	}
	// 正式环境需注释掉
	//localMode("192.168.16.88", "59999", "127.0.0.1")

	//第四步：启动Export
	helper.Export = edgex.NewEdgexExport()
	helper.Export.Init()
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
