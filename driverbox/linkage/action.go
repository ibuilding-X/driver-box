package linkage

// ActionType 执行动作类型
type ActionType string

const (
	// ActionTypeDevicePoint 执行类型：设置设备点位
	ActionTypeDevicePoint ActionType = "devicePoint"
	// ActionTypeLinkage 执行类型：触发场景联动
	ActionTypeLinkage ActionType = "linkage"
)

type Action struct {
	Type ActionType `json:"type" validate:"required,oneof=devicePoint linkage"`
	// Condition 执行条件
	Condition []Condition `json:"condition" validate:"omitempty"`
	// Sleep 执行后休眠时长
	Sleep string `json:"sleep" validate:"omitempty"`
	DevicePointAction
	SceneAction
}

// DevicePointAction 设备点位动作
type DevicePointAction struct {
	// DeviceID 设备 ID
	DeviceID string `json:"devSn" validate:"omitempty"`
	// Points 支持批量设置多个点位值
	Points []DevicePoint `json:"points" validate:"omitempty"`
}

// DevicePoint 设备点位动作项
type DevicePoint struct {
	Point string `json:"point" validate:"omitempty"`
	Value string `json:"value" validate:"omitempty"`
}

// SceneAction 触发场景联动动作
type SceneAction struct {
	ID string `json:"id" validate:"omitempty"`
}
