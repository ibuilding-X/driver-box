package models

import "encoding/json"

// SendPointValues 点位数据
type SendPointValues struct {
	DeviceName string       `json:"deviceName"`
	Mode       string       `json:"mode"`
	Values     []PointValue `json:"values"`
}

func (s SendPointValues) ToJson() string {
	b, _ := json.Marshal(s)
	return string(b)
}
