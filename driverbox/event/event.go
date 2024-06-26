package event

const (
	//设备在离线状态事件
	EventCodeDeviceStatus = "deviceStatus"
	//driver-box服务状态
	EventCodeServiceStatus = "serviceStatus"
)

const (
	//服务启动成功
	ServiceStatusHealthy = "healthy"
	//服务启动异常
	ServiceStatusError = "error"
)

// Data 设备事件模型
type Data struct {
	Code  string      `json:"code"` //事件Code
	Value interface{} `json:"value"`
}
