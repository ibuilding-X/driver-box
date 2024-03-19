package response

// Common 通用 json 响应
type Common struct {
	Message string `json:"message"`
	Data    any    `json:"data"`
}
