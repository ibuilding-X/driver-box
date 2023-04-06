package models

// SendRequest 发送请求数据
type SendRequest struct {
	// Type 请求类型：read、write
	Type string `json:"type"`
	// 设备名称
	DeviceName string `json:"device_name"`
	// 相关点位信息
	PointValues []PointValue `json:"point_values"`
}
