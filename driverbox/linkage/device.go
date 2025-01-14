package linkage

type (
	// DeviceReader 设备点位读取器，用于读取设备指定点位值，与配置条件中的表达式进行比较
	DeviceReader func(deviceID string, point string) (interface{}, error)
	// DeviceWriter 设备点位写入器，用于触发器对设备下发控制
	DeviceWriter func(deviceID string, points []DevicePoint) (err error)
)

type deviceReadWriter interface {
	Read(deviceID string, point string) (interface{}, error)
	Write(deviceID string, points []DevicePoint) (err error)
}

type Device struct {
	// ID 设备ID
	DeviceID string
	// Points 设备点位
	Points []DevicePoint
}

type deviceManager struct {
	readHandler  DeviceReader
	writeHandler DeviceWriter
}

func (d *deviceManager) Read(deviceID string, point string) (interface{}, error) {
	return d.readHandler(deviceID, point)
}

func (d *deviceManager) Write(deviceID string, points []DevicePoint) (err error) {
	return d.writeHandler(deviceID, points)
}
