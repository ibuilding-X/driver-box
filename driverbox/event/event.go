package event

const (
	//设备在离线状态事件
	EventCodeDeviceStatus = "deviceStatus"
	//driver-box服务状态
	EventCodeServiceStatus = "serviceStatus"
	//添加设备
	EventCodeAddDevice = "addDevice"
	//删除设备,在真正之前删除操作之前触发
	EventCodeDeleteDevice = "deleteDevice"
	//即将执行ExportTo
	EventCodeWillExportTo = "willExportTo"
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
