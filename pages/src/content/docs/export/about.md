---
title: Export概要
description: A guide in my new Starlight docs site.
---


Export 模块是 driver-box 框架中的核心功能模块，主要用于将设备数据和事件导出至外部系统。通过 Export，用户可以实现以下功能：

- **数据上云：**
将设备数据实时发送至云平台，便于在云端进行数据存储、分析和展示。

- **场景联动：**
根据设备数据的变化触发特定的场景联动。例如：
  - 当温度传感器检测到温度超过阈值时，自动开启空调。
  - 当烟雾传感器触发告警时，自动关闭电源并通知物业。

- **边缘计算：**
在边缘端实现数据的初步处理和分析，减少对云端的依赖。例如：
  - 实时计算设备的能耗数据。
  - 根据设备状态动态调整控制策略。

- **设备事件通知：**

    将设备的在线/离线状态、告警事件等信息通知至第三方系统。例如：
  - 当设备离线时，通知运维人员。
  - 当设备触发告警时，发送通知至手机端。

Export 的设计目标是提供一个灵活且可扩展的机制，使用户能够根据实际需求定义数据导出逻辑。

---


## 接口设计

Export 模块的核心接口设计如下：

### 1. Export 接口
Export 接口定义了导出模块的核心功能：

```go
type Export interface {
    // 初始化导出模块
    Init() error
    
    // 导出设备数据
    ExportTo(deviceData DeviceData)
    
    // 事件触发回调
    OnEvent(eventCode string, key string, eventValue interface{}) error
    
    // 检查导出模块是否就绪
    IsReady() bool
}
```

#### 方法说明
- `Init()`: 初始化导出模块，配置导出目标和参数。
- `ExportTo(deviceData DeviceData)`: 将设备数据导出至目标系统。
- `OnEvent(eventCode string, key string, eventValue interface{})`: 处理设备事件，例如设备在线/离线状态变化。
- `IsReady()`: 检查导出模块是否已经初始化完成。

### 2. DeviceData 结构
`DeviceData` 结构用于封装设备数据和事件信息：

```go
type DeviceData struct {
    ID      string                 `json:"id"`      // 设备 ID
    Points  map[string]interface{} `json:"points"`  // 设备点位数据
    Events  []Event                `json:"events"`  // 设备事件
    Online  bool                  `json:"online"`  // 设备在线状态
}
```

### 3. 常见 Export 实现
driver-box 提供了多种内置的 Export 实现，用户也可以根据需求自定义 Export：

- **MQTT Export**: 将设备数据通过 MQTT 协议发送至 MQTT 代理。
- **HTTP Export**: 将设备数据通过 HTTP 请求发送至指定 URL。
- **场景联动 Export**: 根据设备数据的变化触发预定义的场景联动逻辑。

---

## 基本使用方式
Export 模块的使用需要结合具体的业务需求，通过实现 `Export` 接口来定义导出逻辑。以下是一个完整的使用流程：

---

### 1. 定义导出模块

首先，需要根据实际需求创建一个 `Export` 接口的实现类。例如，实现一个 MQTT 导出模块：

```go
package myexport

import (
	"github.com/ibuilding-x/driver-box/driverbox/export"
	"github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"
)

type MqttExport struct {
	Broker    string `json:"broker"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	ClientID  string `json:"client_id"`
	Topic     string `json:"topic"`
	client    mqtt.Client
	init      bool
}

func (export *MqttExport) Init() error {
	// 初始化 MQTT 客户端
	options := mqtt.NewClientOptions()
	options.AddBroker(export.Broker)
	options.SetUsername(export.Username)
	options.SetPassword(export.Password)
	options.SetClientID(export.ClientID)

	export.client = mqtt.NewClient(options)
	token := export.client.Connect()
	if token.WaitTimeout(5*time.Second) && token.Error() != nil {
		return token.Error()
	}
	export.init = true
	return nil
}

func (export *MqttExport) ExportTo(deviceData export.DeviceData) {
	// 将设备数据序列化为 JSON
	data, _ := json.Marshal(deviceData)
	// 发布消息至 MQTT 主题
	export.client.Publish(export.Topic, 0, false, data)
}

func (export *MqttExport) OnEvent(eventCode string, key string, eventValue interface{}) error {
	// 处理事件逻辑
	return nil
}

func (export *MqttExport) IsReady() bool {
	return export.init
}
```

---

### 2. 注册导出模块

在 `main.go` 中注册自定义的 Export 模块：

```go
package main

import (
	"github.com/ibuilding-x/driver-box/driverbox/export"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/driverbox/restful"
	"os"
	"time"
)

func main() {
	// 关闭 driver-box 日志
	os.Setenv("LOG_LEVEL", "error")

	// 注册自定义 Export
	export := &myexport.MqttExport{
		Broker:    "tcp://mqtt.example.com:1883",
		Username:  "admin",
		Password:  "password",
		ClientID:  "driver-box",
		Topic:     "device/data",
	}
	export0.Exports = append(export0.Exports, export)

	// 启动 driver-box 服务
	driverbox.Start([]export.Export{})

	// 启动定时任务（可选）
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				// 自定义逻辑
			}
		}
	}()
	select {}
}
```

---

### 3. 配置导出目标

通过 JSON 配置文件或代码配置导出目标。例如，通过代码配置 MQTT 导出模块：

```go
export := &myexport.MqttExport{
	Broker:    "tcp://mqtt.example.com:1883",
	Username:  "admin",
	Password:  "password",
	ClientID:  "driver-box",
	Topic:     "device/data",
}
export0.Exports = append(export0.Exports, export)
```

---

### 4. 启动服务

启动 driver-box 服务后，Export 模块会自动处理设备数据和事件的导出逻辑：

```go
driverbox.Start([]export.Export{})
```

---

## 注意事项

1. **导出模块初始化**：
    - 确保 `Init()` 方法正确配置导出目标，例如 MQTT 代理地址、HTTP 请求目标 URL 等。
    - 初始化失败会导致导出模块无法使用。

2. **数据格式**：
    - `ExportTo()` 方法中的 `deviceData` 需要根据目标系统的要求进行格式化处理。
    - 确保数据序列化（如 JSON）正确无误。

3. **事件处理**：
    - `OnEvent()` 方法需要高效处理事件，避免阻塞主线程。
    - 根据实际需求定义事件处理逻辑。

4. **性能优化**：
    - 对于高频率的数据导出场景，建议优化导出逻辑，避免性能瓶颈。
    - 使用异步处理机制提升导出效率。

---

## 总结
Export 模块是 driver-box 框架中不可或缺的一部分，它为用户提供了一个灵活且可扩展的机制，用于将设备数据和事件导出至外部系统。通过 Export，用户可以轻松实现数据上云、场景联动、边缘计算等功能，从而构建出一个高效、智能的物联网系统。