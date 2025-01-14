package linkage

import (
	"errors"
)

var (
	// ErrAlreadyExist 场景已存在错误
	ErrAlreadyExist = errors.New("linkage already exist")
	// ErrNotFound 场景不存在错误
	ErrNotFound = errors.New("linkage not found")
	// ErrActionEmpty 场景动作为空错误
	ErrActionEmpty = errors.New("linkage action is empty")
	// ErrUnknownTriggerType 未知触发器类型错误
	ErrUnknownTriggerType = errors.New("linkage unknown trigger type")
	// ErrTooDeep 场景层级过深错误
	ErrTooDeep = errors.New("linkage is too deep")
	// ErrExecTooFast 场景执行太快错误
	ErrExecTooFast = errors.New("linkage exec too fast")
	// ErrUnknownConditionType 未知条件类型错误
	ErrUnknownConditionType = errors.New("linkage unknown condition type")
)

// Linkage 场景联动
type Linkage interface {
	// Add 添加场景
	Add(config Config) error
	// Update 更新场景
	Update(config Config) error
	// Delete 删除场景
	Delete(id string) error
	// Trigger 触发场景
	Trigger(id string) error
	// All 返回所有场景配置
	All() []Config
	// QueryByID 方法根据 ID 查询对应的配置项
	QueryByID(id string) (Config, error)
	// QueryByTag 方法根据 Tag 查询对应的配置项
	QueryByTag(tag string) ([]Config, error)
	// DevicePointsBus 设备点位总线，应用调用该方法将点位数据通知场景实例，实例根据点位数据触发场景
	DevicePointsBus(device Device)
}

func New(options *Options) Linkage {
	return &service{
		deviceReadWriter:   options.deviceManager,
		execResultCallback: options.callback,
	}
}

func NewOptions() *Options {
	return &Options{
		deviceManager: &deviceManager{},
	}
}

type Options struct {
	deviceManager *deviceManager
	callback      Callback
}

func (o *Options) SetDeviceReader(reader DeviceReader) {
	o.deviceManager.readHandler = reader
}

func (o *Options) SetDeviceWriter(writer DeviceWriter) {
	o.deviceManager.writeHandler = writer
}

func (o *Options) SetCallback(callback Callback) {
	o.callback = callback
}
