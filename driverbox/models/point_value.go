package models

type PointValue struct {
	// PointName 点位名称
	PointName string `json:"pointName"`
	// Value 点位值
	Value interface{} `json:"value"`
	//模型名称，某些驱动解析需要根据模型作区分
	ModelName string `json:"modelName"`
	//前置操作，例如空开要先解锁，空调要先开机
	PreOp []PointValue `json:"preOp"`
}
