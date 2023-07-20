package common

import "errors"

const (
	CoreConfigPath = "./driver-config" // 本地核心配置文件路径
	LuaScriptName  = "converter.lua"   // lua 转换器脚本名称
	CoreConfigName = "config.json"     // 核心配置文件名称
)

// AdapterScriptPath 协议适配器脚本路径
const AdapterScriptPath = "./driver-config/converter.lua"

var (
	InitLoggerErr                       = errors.New("init logger error")                                      // 初始化日志记录器错误
	NotSupportGetConnector              = errors.New("the protocol does not support getting connector")        // 协议不支持获取连接器
	NotSupportEncode                    = errors.New("the protocol adapter does not support encode functions") // 协议不支持编码函数
	NotSupportDecode                    = errors.New("the protocol adapter does not support decode functions") // 协议不支持解码函数
	ProtocolDataFormatErr               = errors.New("protocol data format error")                             // 协议数据格式错误
	LoadCoreConfigErr                   = errors.New("load core config error")                                 // 加载核心配置文件错误
	ConnectorNotFound                   = errors.New("connector not found error")                              // 连接未找到错误
	NotSupportMode                      = errors.New("not support mode error")                                 // 不支持的模式
	UnsupportedWriteCommandRegisterType = errors.New("unsupport write command register type")                  // 不支持写的寄存器类型
	DeviceNotFoundError                 = errors.New("device not found error")                                 // 设备未找到
	PointNotFoundError                  = errors.New("point not found error")                                  // 点位未找到
)

const (
	ValueTypeBool         = "Bool"
	ValueTypeString       = "String"
	ValueTypeUint8        = "Uint8"
	ValueTypeUint16       = "Uint16"
	ValueTypeUint32       = "Uint32"
	ValueTypeUint64       = "Uint64"
	ValueTypeInt8         = "Int8"
	ValueTypeInt16        = "Int16"
	ValueTypeInt32        = "Int32"
	ValueTypeInt64        = "Int64"
	ValueTypeFloat32      = "Float32"
	ValueTypeFloat64      = "Float64"
	ValueTypeBinary       = "Binary"
	ValueTypeBoolArray    = "BoolArray"
	ValueTypeStringArray  = "StringArray"
	ValueTypeUint8Array   = "Uint8Array"
	ValueTypeUint16Array  = "Uint16Array"
	ValueTypeUint32Array  = "Uint32Array"
	ValueTypeUint64Array  = "Uint64Array"
	ValueTypeInt8Array    = "Int8Array"
	ValueTypeInt16Array   = "Int16Array"
	ValueTypeInt32Array   = "Int32Array"
	ValueTypeInt64Array   = "Int64Array"
	ValueTypeFloat32Array = "Float32Array"
	ValueTypeFloat64Array = "Float64Array"
	ValueTypeObject       = "Object"
)
