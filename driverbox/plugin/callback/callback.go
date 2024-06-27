package callback

import (
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/internal/export"
	"go.uber.org/zap"
)

// 插件收到通讯消息后，触发该回调方法进行消息解码和设备数据解析
func OnReceiveHandler(connector plugin.Connector, raw interface{}) (result interface{}, err error) {
	helper.Logger.Debug("raw data", zap.Any("data", raw))
	// 协议适配器
	deviceData, err := connector.ProtocolAdapter().Decode(raw)
	if err != nil {
		return nil, err
	}
	ExportTo(deviceData)
	return
}

func ExportTo(deviceData []plugin.DeviceData) {
	helper.Logger.Debug("export data", zap.Any("data", deviceData))
	// 写入消息总线
	for _, data := range deviceData {
		helper.PointCacheFilter(&data)
		//触发事件通知
		if len(data.Events) > 0 {
			for _, event := range data.Events {
				export.TriggerEvents(event.Code, data.ID, event.Value)
			}
		}
		if len(data.Values) == 0 {
			continue
		}
		for _, export := range export.Exports {
			if export.IsReady() {
				export.ExportTo(data)
			}
		}
	}
}
