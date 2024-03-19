package request

type UpdateDevicePointsReq []UpdateDevicePointsData

// UpdateDevicePointsData 更新设备点位请求数据
type UpdateDevicePointsData struct {
	Point string `json:"point"`
	Value any    `json:"value"`
}
