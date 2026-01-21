package driverbox

import (
	"github.com/ibuilding-x/driver-box/internal/logger"
	"go.uber.org/zap"
)

func Log() *zap.Logger {
	return logger.Logger
}
