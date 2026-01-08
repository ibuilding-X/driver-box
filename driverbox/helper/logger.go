package helper

import (
	"github.com/ibuilding-x/driver-box/driverbox/internal/logger"
	"go.uber.org/zap"
)

// Logger 日志记录器
var Logger *zap.Logger

// New 实例化
func InitLogger(level string) (err error) {
	logger.InitLogger(EnvConfig.LogPath, level)
	Logger = logger.Logger
	return err
}
