package common

import (
	"errors"
	"io"
	"os"

	"github.com/ibuilding-x/driver-box/driverbox/pkg/event"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
)

var (
	InitLoggerErr                       = errors.New("init logger error")                                      // 初始化日志记录器错误
	NotSupportGetConnector              = errors.New("the protocol does not support getting connector")        // 协议不支持获取连接器
	NotSupportEncode                    = errors.New("the protocol adapter does not support encode functions") // 协议不支持编码函数
	NotSupportDecode                    = errors.New("the protocol adapter does not support decode functions") // 协议不支持解码函数
	ProtocolDataFormatErr               = errors.New("protocol data format error")                             // 协议数据格式错误
	LoadCoreConfigErr                   = errors.New("load core config error")                                 // 加载核心配置文件错误
	ConnectorNotFound                   = errors.New("connector not found error")                              // 连接未找到错误
	NotSupportMode                      = errors.New("not support mode error")                                 // 不支持的模式
	UnsupportedWriteCommandRegisterType = errors.New("unsupport write command register type")                  // 不支持写的寄存器类型
	DeviceNotFoundError                 = errors.New("device not found error")                                 // 设备未找到
	PointNotFoundError                  = errors.New("point not found error")                                  // 点位未找到
)

// FileExists 判断文件存在
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// 读取文件内容
func ReadFileBytes(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return io.ReadAll(f)
}

// 封装设备自动发现事件，补充必要字段
func WrapperDiscoverEvent(devicesData []plugin.DeviceData, connectionKey string, protocolName string) {
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
