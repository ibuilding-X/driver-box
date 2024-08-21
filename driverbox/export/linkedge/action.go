package linkedge

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
	DevicePointAction
	SceneAction
}

// DevicePointAction 设备点位动作
type DevicePointAction struct {
	// DeviceID 设备 ID
	DeviceID string `json:"devSn"`
	// DevicePoint 点位名称
	DevicePoint string `json:"point"`
	// Value 值
	Value string `json:"value"`
}

// SceneAction 触发场景联动动作
type SceneAction struct {
	ID string `json:"id"`
}
