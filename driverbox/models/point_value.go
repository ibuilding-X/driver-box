package models

type PointValue struct {
	// PointName 点位名称
	PointName string `json:"pointName"`
	// Value 点位值
	Value interface{} `json:"value"`
}
