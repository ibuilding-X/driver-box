package export

import (
	"github.com/ibuilding-x/driver-box/v2/driverbox/export"
	"github.com/ibuilding-x/driver-box/v2/internal/logger"
	"github.com/ibuilding-x/driver-box/v2/pkg/event"
	"go.uber.org/zap"
)

var Exports []export.Export

func TriggerEvents(eventCode event.EventCode, key string, value interface{}) {
	for _, e := range Exports {
		if !e.IsReady() {
			logger.Logger.Debug("export not ready")
			continue
		}
		err := e.OnEvent(eventCode, key, value)
		if err != nil {
			logger.Logger.Error("trigger event error", zap.Any("eventCode", eventCode), zap.String("key", key), zap.Any("value", value), zap.Error(err))
		}
	}
}
