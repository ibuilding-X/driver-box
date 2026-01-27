package response

// Common 通用 json 响应
type Common struct {
	Success   bool   `json:"success"`   // 是否成功
	ErrorCode int    `json:"errorCode"` // 错误码
	ErrorMsg  string `json:"errorMsg"`  // 错误信息
	Data      any    `json:"data"`      // 数据
}
