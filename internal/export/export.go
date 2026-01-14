package export

import (
	"github.com/ibuilding-x/driver-box/driverbox/export"
	"github.com/ibuilding-x/driver-box/internal/logger"
	"go.uber.org/zap"
)

var Exports []export.Export

func TriggerEvents(eventCode string, key string, value interface{}) {
	for _, e := range Exports {
		if !e.IsReady() {
			logger.Logger.Debug("export not ready")
			continue
		}
		err := e.OnEvent(eventCode, key, value)
		if err != nil {
			logger.Logger.Error("trigger event error", zap.String("eventCode", eventCode), zap.String("key", key), zap.Any("value", value), zap.Error(err))
		}
	}
}
