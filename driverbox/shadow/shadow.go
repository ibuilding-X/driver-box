package shadow

import (
	"time"
)

// Device 设备影子数据结构
// 用于存储设备的完整状态信息,包括在线状态、点位值、连接统计等
type Device struct {
	// ID 设备唯一标识符
	ID string `json:"id"`

	// ModelName 设备模型名称,对应设备定义的模型类型
	ModelName string `json:"modelName"`

	// Points 设备点位数据集合,key为点位名称,value为点位详细信息
	Points map[string]DevicePoint `json:"points"`

	// Online 设备在线状态,true表示在线,false表示离线
	Online bool `json:"online"`

	// TTL 设备状态存活时间,超过此时间未更新则认为设备可能离线
	TTL string `json:"ttl"`

	// DisconnectTimes 设备累计断开连接次数,用于监控设备稳定性
	DisconnectTimes int `json:"disconnectTimes"`

	// UpdatedAt 设备状态最后更新时间
	UpdatedAt time.Time `json:"updatedAt"`
}

// DevicePoint 设备点位影子数据结构
// 用于存储单个点位的值、写入值及时间戳信息
type DevicePoint struct {
	// Name 点位名称,与设备模型中定义的点位名称一致
	Name string `json:"name"`

	// Value 点位当前值,表示设备上报的最新值
	Value interface{} `json:"value"`

	// WriteValue 最近一次下发给设备的控制值
	WriteValue interface{} `json:"writeValue"`

	// UpdatedAt 点位值最后更新时间
	UpdatedAt time.Time `json:"updatedAt"`

	// WriteAt 写入值最后更新时间
	WriteAt time.Time `json:"writeAt"`
}

// DeviceShadow 设备影子
// 设备影子服务用于维护设备的最新状态，提供状态缓存和监控功能
// 支持设备状态持久化、在线状态跟踪、点位值缓存等特性
type DeviceShadow interface {
	// AddDevice 新增设备到影子服务
	// 参数:
	//   id: 设备唯一标识符
	//   modelName: 设备模型名称
	//   ttl: 设备状态存活时间，可选参数，默认使用系统配置
	//
	// 此方法创建一个新的设备影子实例，用于跟踪设备状态
	AddDevice(id string, modelName string, ttl ...time.Duration)
	// GetDevice 获取设备完整信息
	// 参数:
	//   id: 设备唯一标识符
	// 返回值:
	//   Device: 设备信息结构体
	//   bool: 设备是否存在
	//
	// 注意: 如果仅需获取ModelName，建议使用CoreCache接口，此方法内部转换开销较大
	GetDevice(id string) (device Device, ok bool)
	// HasDevice 检查设备是否存在
	// 参数:
	//   id: 设备唯一标识符
	// 返回值:
	//   bool: 设备是否存在
	HasDevice(id string) bool
	// DeleteDevice 从影子服务删除设备
	// 参数:
	//   id: 设备唯一标识符，支持批量删除
	// 返回值:
	//   error: 删除过程中发生的错误
	DeleteDevice(id ...string) (err error)

	// SetDevicePoint 设置设备点位值
	// 参数:
	//   id: 设备唯一标识符
	//   pointName: 点位名称
	//   value: 点位值
	// 返回值:
	//   error: 设置过程中发生的错误
	//
	// 此方法更新设备点位的当前值并记录更新时间
	SetDevicePoint(id, pointName string, value interface{}) (err error)
	// GetDevicePoint 获取设备点位值
	// 参数:
	//   id: 设备唯一标识符
	//   pointName: 点位名称
	// 返回值:
	//   interface{}: 点位当前值
	//   error: 获取过程中发生的错误
	GetDevicePoint(id, pointName string) (value interface{}, err error)
	// GetDevicePoints 获取设备所有点位
	// 参数:
	//   id: 设备唯一标识符
	// 返回值:
	//   map[string]DevicePoint: 设备所有点位的映射
	//   error: 获取过程中发生的错误
	GetDevicePoints(id string) (points map[string]DevicePoint, err error)
	// GetDevicePointDetails 获取设备点位详细信息
	// 参数:
	//   id: 设备唯一标识符
	//   pointName: 点位名称
	// 返回值:
	//   DevicePoint: 包含点位值、更新时间等详细信息的结构体
	//   error: 获取过程中发生的错误
	GetDevicePointDetails(id, pointName string) (point DevicePoint, err error)

	// IsOnline 获取设备在线状态
	// 参数:
	//   id: 设备唯一标识符
	// 返回值:
	//   bool: 设备是否在线
	//   error: 获取过程中发生的错误
	IsOnline(id string) (online bool, err error)
	// SetOnline 设置设备为在线状态
	// 参数:
	//   id: 设备唯一标识符
	// 返回值:
	//   error: 设置过程中发生的错误
	SetOnline(id string) (err error)
	// SetOffline 设置设备为离线状态
	// 参数:
	//   id: 设备唯一标识符
	// 返回值:
	//   error: 设置过程中发生的错误
	SetOffline(id string) (err error)

	// MayBeOffline 触发可能离线事件
	// 当设备在短时间内多次无法通信时调用此方法
	// 参数:
	//   id: 设备唯一标识符
	// 返回值:
	//   error: 处理过程中发生的错误
	//
	// 此方法实现设备离线检测逻辑，如60秒内超过3次通信失败则标记为离线
	MayBeOffline(id string) (err error)

	// GetDevices 获取所有设备
	// 返回值:
	//   []Device: 所有设备的切片
	//
	// 此方法返回当前影子服务管理的所有设备信息
	GetDevices() []Device

	// SetWritePointValue 存储下发控制点位值
	// 参数:
	//   id: 设备唯一标识符
	//   pointName: 点位名称
	//   value: 要写入的值
	// 返回值:
	//   error: 存储过程中发生的错误
	//
	// 此方法用于存储向设备下发的控制指令值，便于追踪控制历史
	SetWritePointValue(id string, pointName string, value interface{}) (err error)
	// GetWritePointValue 获取下发控制点位值
	// 参数:
	//   id: 设备唯一标识符
	//   pointName: 点位名称
	// 返回值:
	//   interface{}: 最近一次下发的控制值
	//   error: 获取过程中发生的错误
	GetWritePointValue(id string, pointName string) (value interface{}, err error)
}
