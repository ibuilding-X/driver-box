// 插件接口

package plugin

import (
	"errors"

	"github.com/ibuilding-x/driver-box/pkg/config"
	"github.com/ibuilding-x/driver-box/pkg/event"
)

var (
	// NotSupportGetConnector 协议不支持获取连接器时返回的错误
	NotSupportGetConnector = errors.New("the protocol does not support getting connector")

	// NotSupportEncode 协议适配器不支持编码功能时返回的错误
	NotSupportEncode = errors.New("the protocol adapter does not support encode functions")

	// NotSupportDecode 协议适配器不支持解码功能时返回的错误
	NotSupportDecode = errors.New("the protocol adapter does not support decode functions")
)

// Plugin 驱动插件接口
// 所有驱动插件都需要实现此接口以集成到driver-box框架中
// 该接口定义了插件的基本生命周期管理和设备通信功能
type Plugin interface {
	// Initialize 初始化插件
	// 该方法在插件启动时被调用，用于执行必要的初始化操作
	// 参数:
	//   - c: 设备配置信息，包含设备连接参数、协议设置等
	//
	// 初始化内容通常包括:
	//   - 连接配置
	//   - 回调函数注册
	//   - 内部数据结构初始化
	Initialize(c config.DeviceConfig)

	// Connector 获取指定设备的连接器
	// 该方法返回一个连接器实例，用于与特定设备进行通信
	// 参数:
	//   - deviceId: 设备唯一标识符
	// 返回值:
	//   - Connector: 设备连接器实例
	//   - error: 获取连接器过程中发生的错误
	//
	// 注意: 不是所有协议都支持此功能，不支持时返回NotSupportGetConnector错误
	Connector(deviceId string) (connector Connector, err error)

	// Destroy 销毁插件并释放资源
	// 该方法在插件停止时被调用，用于执行清理操作
	// 返回值:
	//   - error: 销毁过程中发生的错误
	//
	// 清理内容通常包括:
	//   - 关闭连接
	//   - 停止协程
	//   - 释放内存资源
	Destroy() error
}

// Connector 设备连接器接口
// 连接器负责与单个设备进行实际的通信操作
// 定义了编码、发送和资源释放等功能
type Connector interface {
	// Encode 编码设备操作指令
	// 将读/写操作和点位数据编码为协议特定的数据格式
	// 参数:
	//   - deviceId: 设备唯一标识符
	//   - mode: 操作模式(读/写)
	//   - values: 点位数据数组，可变参数，支持批量操作
	// 返回值:
	//   - interface{}: 编码后的数据，具体格式取决于协议实现
	//   - error: 编码过程中发生的错误
	//
	// 注意: 不是所有协议都支持编码功能，不支持时返回NotSupportEncode错误
	Encode(deviceId string, mode EncodeMode, values ...PointData) (res interface{}, err error)

	// Send 发送编码后的数据到设备
	// 将编码后的数据通过底层通信机制发送到目标设备
	// 参数:
	//   - data: 通过Encode方法编码后的数据
	// 返回值:
	//   - error: 发送过程中发生的错误
	Send(data interface{}) (err error)

	// Release 释放连接器占用的资源
	// 该方法用于清理连接器相关资源，如关闭连接、停止监听等
	// 返回值:
	//   - error: 释放过程中发生的错误
	//
	// 注意: 调用此方法后，连接器将不再可用
	Release() (err error)
}

// WrapperDiscoverEvent 封装设备自动发现事件，补充必要字段
// 该函数为设备发现事件添加协议和连接相关信息
// 参数:
//   - devicesData: 包含设备数据和事件的数组
//   - connectionKey: 连接标识符
//   - protocolName: 协议名称
//
// 功能:
//   - 遍历设备数据中的事件
//   - 仅处理设备发现事件(event.DeviceDiscover)
//   - 为事件数据补充connectionKey和protocolName字段
func WrapperDiscoverEvent(devicesData []DeviceData, connectionKey string, protocolName string) {
	for _, device := range devicesData {
		if device.Events == nil || len(device.Events) == 0 {
			continue
		}
		for _, eventData := range device.Events {
			//补充信息要素
			if eventData.Code != event.DeviceDiscover {
				continue
			}
			value := eventData.Value.(map[string]interface{})
			value["connectionKey"] = connectionKey
			value["protocolName"] = protocolName
		}
	}
}
