package models

// ScriptResult Lua脚本返回数据格式
type ScriptResult struct {
	DeviceName  string       `json:"deviceName"`
	PointValues []PointValue `json:"pointValues"`
}
