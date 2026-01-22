package route

const V1Prefix string = "/api/v1/"

// 设备写操作
const DevicePointWrite = V1Prefix + "device/writePoint"

// 批量写入某个设备的点位
const DevicePointsWrite = V1Prefix + "device/writePoints"

const DevicePointRead = V1Prefix + "device/readPoint"

// 查询设备列表
const DeviceList = V1Prefix + "device/list"

// 获取设备信息
const DeviceGet = V1Prefix + "device/get"

// 删除设备
const DeviceDelete = V1Prefix + "device/delete"
