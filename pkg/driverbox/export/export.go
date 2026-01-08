package export

import (
	"github.com/ibuilding-x/driver-box/pkg/driverbox/plugin"
)

// Export 定义了驱动数据导出的标准接口
// 实现该接口的模块可以将设备数据导出到不同目标(如EdgeX总线、MQTT等)
// 该接口提供了驱动数据导出的核心功能，包括:
// 1. 初始化导出模块
// 2. 设备数据导出
// 3. 事件处理回调
// 4. 状态检查
// 所有导出模块都需要实现此接口才能被driver-box框架加载和使用
type Export interface {
	// Init 初始化导出模块
	// 该方法在导出模块加载时被调用，用于执行必要的初始化操作
	// 如: 建立连接、加载配置、注册路由等
	// 返回值:
	//   error - 初始化过程中发生的错误，成功返回nil
	Init() error

	// ExportTo 导出设备数据
	// 该方法在设备数据发生变化时被调用，将数据推送到配置的目标
	// 参数:
	//   deviceData - 包含设备ID、点位名称和值的设备数据结构
	//      deviceData.ID: 设备唯一标识
	//      deviceData.Values: 点位数据集合
	// 功能:
	//   将设备数据导出到配置的目标(如EdgeX总线、MQTT等)
	//   实现时应注意处理异常情况并记录日志
	ExportTo(deviceData plugin.DeviceData)

	// OnEvent 事件回调接口
	// 当框架触发特定事件时调用此方法
	// 参数:
	//   eventCode - 事件代码，标识事件类型
	//     常见事件类型: 设备发现、场景联动触发等
	//   key - 事件关联的键值
	//     通常是设备ID或场景ID
	//   eventValue - 事件关联的值
	//     事件相关的数据，类型根据事件不同而变化
	// 返回值:
	//   error - 处理事件过程中发生的错误，成功返回nil
	// 功能:
	//   处理特定事件触发的业务逻辑
	//   实现时应根据eventCode进行不同处理
	OnEvent(eventCode string, key string, eventValue interface{}) error

	// IsReady 检查导出模块是否就绪
	// 该方法用于检查导出模块是否已完成初始化并准备好处理数据
	// 返回值:
	//   bool - true表示模块已就绪，false表示未就绪
	// 注意:
	//   框架会在调用ExportTo和OnEvent前检查此状态
	IsReady() bool

	// Destroy 退出服务
	Destroy() error
}
