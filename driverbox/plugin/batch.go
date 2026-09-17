package plugin

import "github.com/ibuilding-x/driver-box/v2/pkg/event"

// MergeDeviceData 合并一次读取或消息中的同设备、同导出类型数据。
// 保留设备首次出现顺序及各设备的点位、事件顺序，忽略无点位且无事件的数据。
// 不合并不同调用，不去重同名点位；返回独立的切片，避免加工点值时修改输入切片。
func MergeDeviceData(data []DeviceData) []DeviceData {
	type key struct {
		id         string
		exportType ExportType
	}
	indexes := make(map[key]int, len(data))
	var batches []DeviceData
	for _, item := range data {
		if len(item.Values) == 0 && len(item.Events) == 0 {
			continue
		}
		k := key{item.ID, item.ExportType}
		index, exists := indexes[k]
		if !exists {
			indexes[k] = len(batches)
			item.Values = append([]PointData(nil), item.Values...)
			item.Events = append([]event.Data(nil), item.Events...)
			batches = append(batches, item)
			continue
		}
		batches[index].Values = append(batches[index].Values, item.Values...)
		batches[index].Events = append(batches[index].Events, item.Events...)
	}
	return batches
}
