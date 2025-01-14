package linkage

type (
	// DeviceReader 设备点位读取器，用于读取设备指定点位值，与配置条件中的表达式进行比较
	DeviceReader func(deviceID string, point string) (interface{}, error)
	// DeviceWriter 设备点位写入器，用于触发器对设备下发控制
	DeviceWriter func(deviceID string, points []DevicePoint) (err error)
)

type deviceReadWriter struct {
	reader DeviceReader
	writer DeviceWriter
}

func newDeviceReadWriter(reader DeviceReader, writer DeviceWriter) *deviceReadWriter {
	return &deviceReadWriter{reader: reader, writer: writer}
}

func (d *deviceReadWriter) Read(deviceID string, point string) (interface{}, error) {
	return d.reader(deviceID, point)
}

func (d *deviceReadWriter) Write(deviceID string, points []DevicePoint) (err error) {
	return d.writer(deviceID, points)
}
