package driverbox

import (
	"github.com/ibuilding-x/driver-box/v2/internal/logger"
	"go.uber.org/zap"
)

// Log 获取日志记录器实例
// 返回系统默认的zap日志记录器，用于记录系统运行时的各种日志信息
// 该记录器已经过初始化配置，可以直接使用
//
// 返回值:
//   - *zap.Logger: 预配置的zap日志记录器实例，支持结构化日志记录
//
// 使用示例:
//
//	driverbox.Log().Info("Service started", zap.String("version", "1.0.0"))
//	driverbox.Log().Error("Failed to connect", zap.Error(err))
//	driverbox.Log().Debug("Debug information", zap.Any("data", data))
func Log() *zap.Logger {
	return logger.Logger
}
