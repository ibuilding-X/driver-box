// Package config 核心配置
package config

import (
	"os"
)

// 环境变量配置项
const (
	//资源文件存放目录
	ENV_RESOURCE_PATH = "DRIVERBOX_RESOURCE_PATH"
	//http服务监听端口号
	ENV_HTTP_LISTEN = "DRIVERBOX_HTTP_LISTEN"

	//UDP discover服务监听端口号
	ENV_UDP_DISCOVER_LISTEN = "DRIVERBOX_UDP_DISCOVER_LISTEN"

	//日志文件存放路径
	ENV_LOG_PATH = "DRIVERBOX_LOG_PATH"

	//日志级别
	ENV_LOG_LEVEL = "LOG_LEVEL"

	//是否虚拟设备模式: true:是,false:否
	ENV_VIRTUAL = "DRIVERBOX_VIRTUAL"

	//是否虚拟设备模式: true:是,false:否
	ENV_LUA_PRINT_ENABLED = "DRIVERBOX_LUA_PRINT_ENABLE"

	//镜像设备功能是否可用
	ENV_EXPORT_MIRROR_ENABLED = "EXPORT_MIRROR_ENABLED"

	//镜像设备功能是否可用
	ENV_EXPORT_DISCOVER_ENABLED = "EXPORT_DISCOVER_ENABLED"

	//MCP功能是否可用
	ENV_EXPORT_MCP_ENABLED = "EXPORT_MCP_ENABLED"

	//设备历史数据存放路径
	EXPORT_HISTORY_DATA_PATH = "EXPORT_HISTORY_DATA_PATH"
	//设备历史数据保存时长，单位（天），默认值：14
	EXPORT_HISTORY_RESERVED_DAYS = "EXPORT_HISTORY_RESERVED_DAYS"
	//剖面数据写入频率，默认值：60s
	EXPORT_HISTORY_SNAPSHOT_FLUSH_INTERVAL = "EXPORT_HISTORY_SNAPSHOT_FLUSH_INTERVAL"
	//实时数据写入频率，默认值：5s
	EXPORT_HISTORY_REAL_TIME_FLUSH_INTERVAL = "EXPORT_HISTORY_REAL_TIME_FLUSH_INTERVAL"
)

// 资源文件目录
var ResourcePath = "./res"

//------------------------------ 设备模型 ------------------------------

//------------------------------ 设备 ------------------------------

// 是否处于虚拟运行模式：未建立真实的设备连接
func IsVirtual() bool {
	return os.Getenv(ENV_VIRTUAL) == "true"
}
