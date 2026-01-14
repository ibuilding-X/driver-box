// 插件接口

package plugin

import (
	"errors"

	"github.com/ibuilding-x/driver-box/pkg/config"
	"github.com/ibuilding-x/driver-box/pkg/event"
)

var (
	NotSupportGetConnector = errors.New("the protocol does not support getting connector")        // 协议不支持获取连接器
	NotSupportEncode       = errors.New("the protocol adapter does not support encode functions") // 协议不支持编码函数
	NotSupportDecode       = errors.New("the protocol adapter does not support decode functions") // 协议不支持解码函数
)

// Plugin 驱动插件
type Plugin interface {
	// Initialize 初始化日志、配置、接收回调
	Initialize(c config.Config)
	// Connector 连接器
	Connector(deviceId string) (connector Connector, err error)
	// Destroy 销毁驱动
	Destroy() error
}

// Connector 连接器
type Connector interface {
	Encode(deviceId string, mode EncodeMode, values ...PointData) (res interface{}, err error) // 编码，是否支持批量的读写操作，由各插件觉得
	Send(data interface{}) (err error)                                                         // 发送数据
	Release() (err error)                                                                      // 释放连接资源
}

// 封装设备自动发现事件，补充必要字段
func WrapperDiscoverEvent(devicesData []DeviceData, connectionKey string, protocolName string) {
	for _, device := range devicesData {
		if device.Events == nil || len(device.Events) == 0 {
			continue
		}
		for _, eventData := range device.Events {
			//补充信息要素
			if eventData.Code != event.EventDeviceDiscover {
				continue
			}
			value := eventData.Value.(map[string]interface{})
			value["connectionKey"] = connectionKey
			value["protocolName"] = protocolName
		}
	}
}
