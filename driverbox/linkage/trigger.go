package linkage

// TriggerType 触发器类型
type TriggerType string

const (
	// TriggerTypeSchedule 事件表触发器
	TriggerTypeSchedule TriggerType = "schedule"
	// TriggerTypeDevicePoint 设备点位触发器
	TriggerTypeDevicePoint TriggerType = "devicePoint"
	// TriggerTypeDeviceEvent 设备事件触发器（暂未使用）
	TriggerTypeDeviceEvent TriggerType = "deviceEvent"
)

// Trigger 触发器
type Trigger struct {
	Type TriggerType `json:"type" validate:"required,oneof=schedule devicePoint deviceEvent"`
	ScheduleTrigger
	DevicePointTrigger
}

// ScheduleTrigger 定时触发器
type ScheduleTrigger struct {
	Cron string `json:"cron" validate:"omitempty,cron"`
}

// DevicePointTrigger 设备点位触发器
type DevicePointTrigger struct {
	// DeviceID 设备 ID
	DeviceID string `json:"devSn" validate:"omitempty"`
	// DevicePoint 点位名称
	DevicePoint string `json:"point" validate:"omitempty"`
	// Condition 条件模式：== != > < 等
	Condition ConditionSymbol `json:"condition" validate:"omitempty,oneof='=' '!=' '>' '>=' '<' '<='"`
	// Value 条件值
	Value string `json:"value" validate:"omitempty"`
}

// DeviceEventTrigger 设备事件触发器
// 提示：暂未使用
type DeviceEventTrigger struct {
}
