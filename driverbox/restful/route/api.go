package route

// 设备写操作
const DevicePointWrite = V1Prefix + "device/writePoint"

const DevicePointRead = V1Prefix + "device/readPoint"

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
