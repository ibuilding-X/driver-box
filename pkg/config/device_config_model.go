package config

// Model 模型基础信息
type Model struct {
	// 模型名称
	Name string `json:"name" validate:"required"`
	// 云端模型 ID
	ModelID string `json:"modelId" validate:"required"`
	// 模型描述
	Description string `json:"description" validate:"required"`
	//扩展属性
	Attributes map[string]interface{} `json:"attributes"`
	// 模型点位列表
	DevicePoints []Point `json:"devicePoints" validate:"required"`
}
