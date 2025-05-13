package event

// todo 后续定义事件code采用 EventCode 类型
type EventCode string

const (
	//设备在离线状态事件
	EventCodeDeviceStatus = "deviceStatus"
	//driver-box服务状态
	EventCodeServiceStatus = "serviceStatus"
	//添加设备
	EventCodeAddDevice = "addDevice"
	//即将删除设备,在该事件中依旧可以查询设备信息
	EventCodeWillDeleteDevice = "willDeleteDevice"
	//即将执行ExportTo
	EventCodeWillExportTo = "willExportTo"
	//设备自动发现事件
	EventDeviceDiscover = "deviceDiscover"

	EventCodeLinkEdgeTrigger = "linkEdgeTrigger"

	// EventCodeOnOff 设备开关事件（空调的开关机、灯的开关……）
	EventCodeOnOff = "onOff"
)

// 场景相关事件
const (
	// UnknownDevice 未知设备
	UnknownDevice = "unknownDevice"
	// UnknownLinkEdge 未知场景
	UnknownLinkEdge = "unknownLinkEdge"
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
