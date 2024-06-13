package request

type UpdateDeviceReq []UpdateDeviceData

// UpdateDeviceData 更新设备点位请求数据
type UpdateDeviceData struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Value any    `json:"value"`
}
