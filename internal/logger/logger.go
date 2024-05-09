package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"os"
	"strings"
	"time"
)

// Logger 日志记录器
var Logger *zap.Logger

// customClock 自定义时钟（时区调整）
type customClock struct {
}

func (c *customClock) Now() time.Time {
	timezone := time.FixedZone("Asia/Shanghai", 8*3600)
	return time.Now().In(timezone)
}

func (c *customClock) NewTicker(duration time.Duration) *time.Ticker {
	return time.NewTicker(duration)
}

// New 实例化
func InitLogger(logPath, level string) {
	config := zap.NewProductionConfig()
	config.Level = convertLevel(level)                                                       // 设置日志级别
	config.Encoding = "console"                                                              // 输出格式：console、json
	config.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000") // 输出时间格式
	config.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder                           // 输出等级格式
	config.EncoderConfig.ConsoleSeparator = " | "                                            // 字段分割符

	var options []zap.Option
	options = append(options, zap.AddCaller())                         // 输出调用者信息
	options = append(options, zap.WithClock(&customClock{}))           // 设置时区
	options = append(options, zap.AddStacktrace(zap.ErrorLevel))       // 错误堆栈信息
	options = append(options, zap.Fields(zap.Int("pid", os.Getpid()))) // 进程ID

	var w io.Writer
	if logPath == "" {
		w = os.Stdout
	} else {
		w = &lumberjack.Logger{
			Filename:   logPath,
			MaxSize:    100,
			MaxAge:     15,
			MaxBackups: 10,
			LocalTime:  true,
			Compress:   true,
		}
	}

	encoder := zapcore.NewConsoleEncoder(config.EncoderConfig)
	writer := zapcore.NewMultiWriteSyncer(zapcore.AddSync(w))
	core := zapcore.NewCore(encoder, writer, config.Level)

	Logger = zap.New(core, options...)
}

// convertLevel 等级转换
func convertLevel(level string) zap.AtomicLevel {
	switch strings.ToLower(level) {
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
