package models

// PointValue 点位值模型
type PointValue struct {
	// PointName 点位名称
	PointName string `json:"pointName"`
	// Type 点位值类型
	Type string `json:"type"`
	// Value 点位值
	Value interface{} `json:"value"`
}
