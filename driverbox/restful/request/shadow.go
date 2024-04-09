package request

type UpdateDeviceReq []UpdateDeviceData

// UpdateDeviceData 更新设备点位请求数据
type UpdateDeviceData struct {
	SN    string `json:"sn"`
	Name  string `json:"name"`
	Value any    `json:"value"`
}
