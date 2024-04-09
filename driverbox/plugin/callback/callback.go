package callback

import (
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"go.uber.org/zap"
)

// 插件收到通讯消息后，触发该回调方法进行消息解码和设备数据解析
func OnReceiveHandler(plugin plugin.Plugin, raw interface{}) (result interface{}, err error) {
	helper.Logger.Debug("raw data", zap.Any("data", raw))
	// 协议适配器
	deviceData, err := plugin.ProtocolAdapter().Decode(raw)
	helper.Logger.Debug("decode data", zap.Any("data", deviceData))
	if err != nil {
		return nil, err
	}
	// 写入消息总线
	for _, data := range deviceData {
		helper.PointCacheFilter(&data)
		if len(data.Values) == 0 {
			continue
		}
		for _, export := range helper.Exports {
			if export.IsReady() {
				export.ExportTo(data)
			}
		}
	}
	return
}
