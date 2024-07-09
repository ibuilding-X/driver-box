package export

import (
	"github.com/ibuilding-x/driver-box/driverbox/export"
	"github.com/ibuilding-x/driver-box/internal/logger"
	"go.uber.org/zap"
)

var Exports []export.Export

// 触发事件
func TriggerEvents(eventCode string, key string, value interface{}) {
	for _, export0 := range Exports {
		if !export0.IsReady() {
			logger.Logger.Debug("export not ready")
			continue
		}
		err := export0.OnEvent(eventCode, key, value)
		if err != nil {
			logger.Logger.Error("trigger event error", zap.String("eventCode", eventCode), zap.String("key", key), zap.Any("value", value), zap.Error(err))
		}
	}
}
