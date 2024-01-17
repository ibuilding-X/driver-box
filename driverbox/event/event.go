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
