# TCP Server 插件改进说明

## 改进概述

基于 WebSocket 插件的设计模式，对 TCP Server 插件进行了全面改进，实现了完整的双向通信和设备生命周期管理。

## 新增功能

### 1. 主动发送能力

```go
// SendToDevice - 主动发送原始数据到指定设备（不经过 Lua encode）
func (c *connector) SendToDevice(deviceId string, data []byte) error

// SendToAll - 广播原始数据到所有已连接设备
func (c *connector) SendToAll(data []byte) (sent int, err error)

// GetConnectedDevices - 获取所有已连接的设备ID列表
func (c *connector) GetConnectedDevices() []string

// IsDeviceConnected - 检查设备是否已连接
func (c *connector) IsDeviceConnected(deviceId string) bool
```

**使用场景：**
- 心跳响应
- 自定义协议命令
- 测试/调试
- 广播通知

### 2. 设备-连接映射管理

```go
// 设备到连接的映射 (deviceId -> net.Conn)
deviceMappingConn *sync.Map

// 连接到设备的映射 (net.Conn -> []string)
connMappingDevice *sync.Map
```

- 自动维护设备与 TCP 连接的双向映射关系
- 支持多设备共享同一连接
- 支持设备连接更新（设备从旧连接切换到新连接）

### 3. 下行数据能力

```go
// Encode - 编码设备操作指令
func (c *connector) Encode(deviceId string, mode plugin.EncodeMode, values ...plugin.PointData) (res interface{}, err error)

// Send - 发送编码后的数据到设备
func (c *connector) Send(raw interface{}) (err error)
```

- 支持通过 Lua 脚本编码下行数据
- 自动查找设备对应的连接
- 支持写操作模式

### 4. 设备离线检测

```go
// cleanupConn - 清理连接断开时的映射
func (c *connector) cleanupConn(conn net.Conn)
```

- 连接断开时自动清理设备映射
- 自动设置设备离线状态
- 触发设备离线事件通知

### 5. 设备自动发现

```go
// 自动发现设备
plugin.WrapperDiscoverEvent(res, c.config.ConnectionKey, ProtocolName)
```

- 支持设备自动发现功能
- 自动补充连接和协议信息

### 6. 连接器获取

```go
// Connector - 获取设备连接器
func (p *Plugin) Connector(deviceId string) (connector plugin.Connector, err error)
```

- 支持通过设备ID获取连接器
- 便于上层模块进行设备操作

## 配置示例

```yaml
plugins:
  - name: tcp_server
    type: tcp_server
    connections:
      ems-connection:
        host: "0.0.0.0"
        port: 5000
        buffSize: 1024
        readTimeout: 30  # 读取超时，单位秒
        enable: true
        protocolKey: "charging_pile_protocol"  # Lua 脚本协议
        discover: true  # 启用设备自动发现
```

## Lua 脚本接口

### decode(raw)

解码上行数据，返回设备数据数组。

**输入参数：**
```json
{
  "remoteAddr": "192.168.1.100:12345",
  "event": "read",
  "raw": "<以 Go string 保存的原始字节>"
}
```

**输出格式：**
```json
[
  {
    "id": "device001",
    "values": [
      {"name": "temperature", "value": 25.5},
      {"name": "humidity", "value": 60}
    ],
    "events": []
  }
]
```

### encode(deviceId, mode, points)

编码下行数据，返回原始数据。

**输入参数：**
- `deviceId`: 设备ID
- `mode`: 操作模式 ("read" 或 "write")
- `points`: 点位数据数组

**输出格式：**
返回编码后的原始数据字符串。

## 测试覆盖

测试文件：`connector_test.go` 和 `plugin_test.go`

覆盖的功能点：
- ✅ 协议数据转换为 JSON
- ✅ 设备映射更新
- ✅ 连接清理和设备离线
- ✅ 编码功能
- ✅ 发送功能
- ✅ 并发映射操作
- ✅ 资源释放
- ✅ 服务器启动
- ✅ 连接处理
- ✅ 编码结构体
- ✅ 连接器配置
- ✅ 插件初始化
- ✅ 连接器获取
- ✅ 插件销毁
- ✅ 多连接配置
- ✅ 禁用连接
- ✅ 并发访问
- ✅ 连接池初始化
- ✅ 无效配置处理
- ✅ 协议名称
- ✅ 基础连接配置
- ✅ 主动发送到指定设备
- ✅ 广播到所有设备
- ✅ 获取已连接设备列表
- ✅ 设备连接状态检查
- ✅ 提取 device_key

## 主动发送使用示例

### 场景1：心跳响应

```go
// conn 是由运行中的 Plugin.Connector(deviceId) 返回的 plugin.Connector
tcpConn, ok := conn.(tcpserver.Connector)
if !ok {
    log.Fatal("not a TCP Server connector")
}

// 构建心跳响应数据
heartbeatResp := []byte{0xAA, 0xF5, 0x00, 0x08, 0x00, 0x01, 0x00, 0x00}

// 主动发送心跳响应
err = tcpConn.SendToDevice("device001", heartbeatResp)
if err != nil {
    log.Printf("heartbeat send failed: %v", err)
}
```

### 场景2：下发控制命令

```go
// conn 是由运行中的 Plugin.Connector(deviceId) 返回的 plugin.Connector
tcpConn, ok := conn.(tcpserver.Connector)
if !ok {
    return
}

// 构建限功率命令（参考充电桩协议）
command := buildPowerLimitCommand("PILE001", 10.0)  // 10KW

// 发送命令
err := tcpConn.SendToDevice("PILE001_gun1", command)
if err != nil {
    log.Printf("command send failed: %v", err)
}
```

### 场景3：广播通知

```go
// conn 是由运行中的 Plugin.Connector(deviceId) 返回的 plugin.Connector
tcpConn, ok := conn.(tcpserver.Connector)
if !ok {
    return
}

// 广播系统时间同步命令
timeSyncCmd := buildTimeSyncCommand()
sent, err := tcpConn.SendToAll(timeSyncCmd)
if err != nil {
    log.Printf("broadcast failed: %v", err)
} else {
    log.Printf("time sync sent to %d devices", sent)
}
```

### 场景4：检查设备状态

```go
// conn 是由运行中的 Plugin.Connector(deviceId) 返回的 plugin.Connector
tcpConn, ok := conn.(tcpserver.Connector)
if !ok {
    return
}

// 获取所有已连接设备
devices := tcpConn.GetConnectedDevices()
log.Printf("connected devices: %v", devices)

// 检查特定设备是否在线
if tcpConn.IsDeviceConnected("device001") {
    log.Println("device001 is online")
} else {
    log.Println("device001 is offline")
}
```

## 与 WebSocket 插件的对比

| 功能 | WebSocket 插件 | TCP Server 插件 (改进后) |
|------|---------------|------------------------|
| 设备-连接映射 | ✅ | ✅ |
| 连接断开检测 | ✅ | ✅ |
| 设备离线通知 | ✅ | ✅ |
| 设备自动发现 | ✅ | ✅ |
| 下行数据 (Send) | ✅ | ✅ |
| 编码 (Encode) | ✅ | ✅ |
| 解码 (Decode) | ✅ | ✅ |
| 事件上下文 | ✅ | ✅ |
| 连接管理 | ✅ | ✅ |
| Lua 脚本集成 | ✅ | ✅ |

## 适用场景

改进后的 TCP Server 插件适用于以下场景：

1. **工业设备通信**
   - 充电桩与 EMS 系统
   - PLC 与 SCADA 系统
   - 智能终端与管理平台

2. **物联网网关**
   - 设备数据采集
   - 远程控制下发
   - 设备状态监控

3. **自定义协议**
   - 二进制协议解析
   - 文本协议处理
   - 混合协议支持

## 设备匹配机制

### 通过 Properties 中的 device_key 匹配设备

当 Lua 返回的设备 ID 与配置中的设备 ID 不一致时，可以通过 Properties 中的 `device_key` 字段进行匹配。

**使用场景：**
- 充电桩上报数据中只有桩编码（如 "PILE001"），没有枪号
- 配置中每个枪是独立的设备（如 "PILE001_gun1", "PILE001_gun2"）
- Lua 脚本需要组合桩编码和枪号生成完整的设备 ID

**设备配置示例：**

```json
{
  "devices": [
    {
      "id": "PILE001_gun1",
      "description": "充电桩1枪1",
      "connectionKey": "ems-server",
      "properties": {
        "device_key": "PILE001_gun1"
      }
    },
    {
      "id": "PILE001_gun2",
      "description": "充电桩1枪2",
      "connectionKey": "ems-server",
      "properties": {
        "device_key": "PILE001_gun2"
      }
    }
  ]
}
```

**Lua 脚本示例：**

```lua
function decode(raw)
    local data = json.decode(raw)
    local raw_data = data["raw"]

    -- 提取桩编码
    local pile_id = string.sub(raw_data, 8, 39):gsub("%z+$", "")

    -- 提取枪号
    local gun_no = string.byte(raw_data, 40)

    -- 组合成完整的设备标识
    local device_key = pile_id .. "_gun" .. gun_no

    return json.encode({
        {
            -- 可以返回组合的 ID
            id = device_key,

            values = {
                -- 返回 device_key 用于匹配
                {name = "device_key", value = device_key},
                -- 其他数据点...
                {name = "status", value = parse_status(raw_data)},
                {name = "voltage", value = parse_voltage(raw_data)}
            }
        }
    })
end
```

**匹配逻辑：**

1. 首先尝试使用返回的 `id` 匹配配置中的设备
2. 如果 `id` 匹配失败，从 `values` 中提取 `device_key`
3. 使用 `device_key` 匹配 Properties 中的 `device_key` 字段
4. 找到匹配的设备后建立连接映射

**匹配示例：**

```
Lua 返回：
{
  id: "PILE001_gun1",
  values: [
    {name: "device_key", value: "PILE001_gun1"},
    {name: "status", value: "charging"}
  ]
}

配置中的设备：
PILE001_gun1: {device_key: "PILE001_gun1"}
PILE001_gun2: {device_key: "PILE001_gun2"}

匹配过程：
1. 尝试匹配 id = "PILE001_gun1" → 找到配置中的设备 → 匹配成功

如果 id 匹配失败：
1. 提取 device_key = "PILE001_gun1"
2. 遍历配置中的设备
3. 查找 Properties.device_key = "PILE001_gun1" 的设备
4. 找到 "PILE001_gun1" → 匹配成功
```

## 示例：充电桩协议

参考 `charging_pile_protocol.lua` 脚本，实现了：

- 二进制协议解析（小端序）
- 校验和验证
- 设备ID提取
- 多类型消息解码（状态上报、命令响应）
- 下行命令编码（限功率命令）
- **通过 device_key 匹配设备**

## 后续改进方向

1. **心跳机制** - 可在 Lua 脚本中实现自定义心跳逻辑
2. **连接池管理** - 支持最大连接数限制
3. **安全认证** - 支持 TLS/SSL 加密连接
4. **性能优化** - 支持批量数据处理
5. **监控指标** - 添加连接数、消息吞吐量等指标
