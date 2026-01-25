package event

// todo 后续定义事件code采用 EventCode 类型
type EventCode string

const (
	//设备在离线状态事件
	DeviceOnline = EventCode("deviceOnline")
	//driver-box服务状态
	ServiceStatus = EventCode("serviceStatus")
	//添加设备
	DeviceAdded = EventCode("deviceAdded")
	//即将删除设备,在该事件中依旧可以查询设备信息
	DeviceDeleting = EventCode("deviceDeleting")
	//即将执行ExportTo
	Exporting = EventCode("exporting")
	//设备自动发现事件
	DeviceDiscover = EventCode("deviceDiscover")

	// DoExport 插件回调事件
	DoExport = EventCode("export")
)

const (
	//服务启动成功
	ServiceStatusHealthy = "healthy"
	//服务启动异常
	ServiceStatusError = "error"
)

// Data 设备事件模型
type Data struct {
	Code  EventCode   `json:"code"` //事件Code
	Value interface{} `json:"value"`
}
