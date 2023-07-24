package helper

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger 日志记录器
var Logger *zap.Logger

// InitLogger 初始化日志记录器
func InitLogger(level string) (err error) {
	en := zap.NewProductionEncoderConfig()
	en.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000")
	en.ConsoleSeparator = " | "
	en.EncodeLevel = zapcore.CapitalLevelEncoder

	conf := zap.NewProductionConfig()
	conf.Level = convLoggerLV(level)
	conf.EncoderConfig = en
	conf.Encoding = "console"
	Logger, err = conf.Build()
	if err != nil {
		return err
	}
	return nil
}

// convLoggerLV 转换日志等级
func convLoggerLV(level string) zap.AtomicLevel {
	switch level {
	case "debug":
		return zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		return zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		return zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		return zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		return zap.NewAtomicLevelAt(zap.DebugLevel)
	}
}
