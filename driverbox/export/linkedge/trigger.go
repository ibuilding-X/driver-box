package linkedge

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
	Type TriggerType `json:"type"`
	ScheduleTrigger
	DevicePointTrigger
}

// ScheduleTrigger 定时触发器
type ScheduleTrigger struct {
	Cron string `json:"cron"`
}

// DevicePointTrigger 设备点位触发器
type DevicePointTrigger struct {
	DevicePointCondition
}

// DeviceEventTrigger 设备事件触发器
// 提示：暂未使用
type DeviceEventTrigger struct {
}
