package model

// ActionType 执行动作类型
type ActionType string

const (
	// ActionTypeDevicePoint 执行类型：设置设备点位
	ActionTypeDevicePoint ActionType = "devicePoint"
	// ActionTypeLinkEdge 执行类型：触发场景联动
	ActionTypeLinkEdge ActionType = "linkEdge"
)

type Action struct {
	Type ActionType `json:"type"`
	// ACondition 执行条件
	Condition []Condition `json:"condition"`
	// Sleep 执行后休眠时长
	Sleep string `json:"sleep"`
	DevicePointAction
	SceneAction
}

// DevicePointAction 设备点位动作
type DevicePointAction struct {
	// DeviceID 设备 ID
	DeviceID string `json:"devSn"`
	// DevicePoint 点位名称（兼容旧版本，后续版本将废弃）
	// Deprecated: 请使用 Points
	DevicePoint string `json:"point"`
	// Value 值（兼容旧版本，后续版本将废弃）
	// Deprecated: 请使用 Points
	Value interface{} `json:"value"`
	// Points 支持批量设置多个点位值
	Points []DevicePointActionItem `json:"points"`
}

// DevicePointActionItem 设备点位动作项
type DevicePointActionItem struct {
	Point string `json:"point"`
	Value string `json:"value"`
}

// SceneAction 触发场景联动动作
type SceneAction struct {
	ID string `json:"id"`
}
