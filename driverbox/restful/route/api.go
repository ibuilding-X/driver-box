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

// 添加设备
const DeviceAdd = V1Prefix + "device/add"

// 删除设备
const DeviceDelete = V1Prefix + "device/delete"

// 创建场景联动
const LinkEdgeCreate = V1Prefix + "linkedge/create"

// 试运行场景，不作持久化
const LinkEdgeTryTrigger = V1Prefix + "linkedge/try"

// 删除场景联动
const LinkEdgeDelete = V1Prefix + "linkedge/delete"

// 触发指定ID的场景联动
const LinkEdgeTrigger = V1Prefix + "linkedge/trigger"

// 获取指定ID的场景联动配置
const LinkEdgeGet = V1Prefix + "linkedge/get"

// 获取场景联动列表
const LinkEdgeList = V1Prefix + "linkedge/list"

// 更新场景联动
const LinkEdgeUpdate = V1Prefix + "linkedge/update"

// 更新场景联动状态
const LinkEdgeStatus = V1Prefix + "linkedge/status"

// deprecated 获取最后一个执行的场景联动
const LinkEdgeGetLast = V1Prefix + "linkedge/getLast"

// modbus驱动--设备发现
const ModbusDeviceDiscovery = V1Prefix + "plugin/modbus/discovery"

// bacnet驱动--设备发现
const BacnetDeviceDiscovery = V1Prefix + "plugin/bacnet/discovery"
