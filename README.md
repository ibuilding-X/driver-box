# driver-box

<div align="center">

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Documentation](https://img.shields.io/badge/docs-latest-green.svg)](https://ibuilding-x.github.io/driver-box/)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/ibuilding-X/driver-box)

一款支持泛化协议接入的轻量级边缘网关框架，通过插件化和Lua脚本实现灵活的设备接入能力。

[快速开始](#快速开始) · [核心特性](#核心特性) · [架构设计](#架构设计) · [二次开发](#二次开发) · [API文档](#api文档)

</div>

---

## 📖 简介

driver-box 是一款专为物联网场景设计的边缘网关框架，采用**插件化架构**融合了 Modbus、BACnet、HTTP、MQTT、WebSocket 等主流工业协议，同时支持基于 TCP 的各类私有协议的无缝对接。

### 设计理念

通过对各类设备的通信协议和数据交互形式进行抽象，定义了一套**标准化的设备接入流程**，涵盖各类通信协议的共性逻辑。同时，结合**动态 Lua 脚本引擎**填补协议差异化的部分，实现了：

- ✅ **解决驱动工程数量爆炸**问题 - 通过配置化和脚本化大幅减少重复开发
- ✅ **统一接入标准** - 建立规范化的设备接入流程
- ✅ **降低技术门槛** - 理想情况下仅需编写 JSON 配置文件即可完成设备接入
- ✅ **灵活扩展能力** - 支持自定义插件和 Lua 脚本应对复杂场景

我们期望 driver-box 能够为 IoT 开发者提供更加高效、舒适的设备接入体验。

---

## 🚀 核心特性

### 💎 轻量高效

- **极简架构**：Go 语言开发，单文件编译，体积控制在 10-20MB
- **低资源占用**：运行内存需求低，满足低规格边缘网关的运行要求
- **跨平台支持**：支持 amd64、arm64、armv7、x86 等多种系统架构
- **高性能并发**：基于 Go 协程实现高效的并发数据处理

### 🔌 插件化架构

- **协议插件**：内置 Modbus、BACnet、HTTP、MQTT、WebSocket 等主流协议支持
- **数据导出**：支持多种数据导出方式（MQTT、HTTP、LinkEdge 等）
- **热插拔**：支持插件动态加载、卸载和配置热重载
- **无限扩展**：开放插件接口，开发者可轻松扩展协议和功能

### 📜 Lua 脚本引擎

- **动态解析**：通过 Lua 脚本实现复杂协议的数据解析和处理
- **业务逻辑**：支持在脚本中编写设备特定的业务逻辑
- **内置库支持**：提供丰富的内置 Lua 库（HTTP、JSON、数学计算等）
- **热更新**：支持运行时更新脚本，无需重启服务

### 🌐 丰富的 API

- **RESTful API**：提供完整的设备管理、数据查询、控制指令等 API
- **WebSocket**：支持实时数据推送和双向通信
- **事件机制**：支持设备事件订阅和自定义事件处理
- **无 UI 依赖**：开放 API，用户可自由定制网关 UI 界面

### 📊 数据管理

- **设备影子**：设备状态缓存和离线数据缓存
- **历史数据**：可选的时序数据存储支持
- **多格式导出**：支持 JSON、Modbus Server 等多种数据格式导出
- **配置管理**：基于文件的配置管理和版本控制

---

## 🏗️ 架构设计

```
┌─────────────────────────────────────────────────────────────┐
│                        Export Layer                         │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────────┐│
│  │ LinkEdge │ │   MQTT   │ │ Gateway  │ │ History/Discover ││
│  └──────────┘ └──────────┘ └──────────┘ └──────────────────┘│
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│                        driver-box Core                       │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────────┐│
│  │  Cache   │ │  Shadow  │ │  Event   │ │   Crontab        ││
│  └──────────┘ └──────────┘ └──────────┘ └──────────────────┘│
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│                       Library Layer                         │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  Driver Scripts (Lua) + Protocol Scripts (Lua)       │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│                      Plugin Layer                           │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────────┐│
│  │  Modbus  │ │  BACnet  │ │   MQTT   │ │ HTTP/WebSocket   ││
│  └──────────┘ └──────────┘ └──────────┘ └──────────────────┘│
└─────────────────────────────────────────────────────────────┘
```

### 目录结构

```
driver-box/
├── driverbox/          # 核心框架
│   ├── driverbox.go    # 主入口
│   ├── plugin.go       # 插件管理
│   ├── export.go       # 导出管理
│   ├── shadow.go       # 设备影子
│   └── crontab.go      # 定时任务
├── plugins/            # 内置协议插件
│   ├── modbus/         # Modbus 协议
│   ├── bacnet/         # BACnet 协议
│   ├── mqtt/           # MQTT 协议
│   ├── httpclient/     # HTTP 客户端
│   ├── httpserver/     # HTTP 服务端
│   ├── tcpserver/      # TCP 服务端
│   ├── websocket/      # WebSocket 协议
│   └── dlt645/         # DLT645 协议
├── exports/            # 数据导出插件
│   ├── linkedge/       # 场景联动
│   ├── mirror/         # 镜像设备服务
│   ├── discover/       # 设备自动发现
│   └── gateway/        # 分布式网关
├── internal/           # 内部实现
│   ├── core/           # 核心逻辑
│   ├── cache/          # 缓存管理
│   ├── export/         # 内置Export
│   └── shadow/         # 影子服务
├── pkg/                # 公共包
│   ├── config/         # 配置管理
│   ├── event/          # 平台事件定义
│   ├── library/        # 资源库
│   └── crontab/        # 定时任务
├── res/                # 资源目录（运行时）
│   ├── library/
│   │   ├── driver/     # 设备驱动脚本
│   │   ├── protocol/   # 协议解析脚本
│   │   └── model/      # 物模型定义
│   ├── computing/      # 计算任务
│   └── history_data/   # 历史数据
├── pages/              # 文档站点
├── main.go             # 应用入口
├── go.mod              # Go 依赖
└── deploy.sh           # 构建脚本
```

---

## 📦 快速开始

### 环境要求

- **Go**: 1.23 或更高版本
- **操作系统**: Linux / Windows / macOS / Android
- **架构**: amd64 / arm64 / armv7 / arm

### 安装

#### 1. 下载源码

```bash
git clone https://github.com/ibuilding-X/driver-box.git
cd driver-box
```

#### 2. 加载依赖

```bash
# 国内用户推荐使用镜像
go env -w GOPROXY=https://goproxy.cn,direct

# 加载依赖
go mod tidy
go mod vendor
```

#### 3. 运行

```bash
# 直接运行
go run main.go

# 或编译后运行
go build -o driver-box
./driver-box
```

### 交叉编译

项目提供了自动化构建脚本，支持多平台交叉编译：

```bash
# 执行构建脚本
bash deploy.sh

# 输出目录
_output/
├── driver-box-linux-arm64-v1.0.0.tar.gz
├── driver-box-linux-amd64-v1.0.0.tar.gz
└── ...
```

### 配置说明

driver-box 使用 `res/` 目录作为配置资源目录，启动时可通过环境变量指定：

```bash
# 默认配置路径
export DRIVERBOX_RESOURCE_PATH="./res"

# 自定义配置路径
export DRIVERBOX_RESOURCE_PATH="/opt/driver-box/res"

# 设置日志级别
export LOG_LEVEL="info"
export DRIVERBOX_LOG_PATH="./logs"
```

---

## 🔌 内置插件

### 协议插件

| 插件名称 | 协议类型 | 说明 |
|---------|---------|------|
| `modbus` | Modbus RTU/TCP | 工业通用协议，支持串口和TCP |
| `bacnet` | BACnet/IP | 楼宇自动化标准协议 |
| `mqtt` | MQTT | 物联网轻量级消息协议 |
| `httpclient` | HTTP Client | HTTP 客户端，支持 REST API |
| `httpserver` | HTTP Server | HTTP 服务端，提供 API 接口 |
| `tcpserver` | TCP Server | TCP 服务端，支持自定义协议 |
| `websocket` | WebSocket | 实时双向通信协议 |
| `dlt645` | DLT645 | 电能表通信协议 |

### export 插件

| 插件名称 | 功能说明               |
|---------|--------------------|
| `linkedge` | 场景联动               |
| `mirror` | 设备数据镜像             |
| `discover` | 设备自动发现服务           |
| `gateway` | 分布式网关              |

### 启用插件

在 `main.go` 中启用需要的插件：

```go
package main

import (
	"os"

	"github.com/ibuilding-x/driver-box/driverbox"
	"github.com/ibuilding-x/driver-box/exports"
	"github.com/ibuilding-x/driver-box/plugins"
)

func main() {
	// 设置日志级别
	_ = os.Setenv("LOG_LEVEL", "info")
	plugins.EnableAll()
	exports.EnableAll()
	driverbox.Start()
	select {}
}

```

---

### 📚 详细开发文档

完整的二次开发文档请参考：

- **[Plugin开发指南](https://ibuilding-x.github.io/driver-box/plugins/development)** - 详细的 Plugin 和 Connector 接口实现指南
- **[Export开发指南](https://ibuilding-x.github.io/driver-box/exports/development)** - 数据导出功能开发教程

---

## 🎯 应用场景

driver-box 适用于多种物联网场景：

| 场景 | 说明 | 支持设备 |
|------|------|---------|
| **智慧楼宇** | HVAC 系统、照明系统、电梯系统 | 空调机组、照明控制器、电表、水表 |
| **智慧工厂** | 生产设备监控、能耗管理 | PLC、变频器、传感器、执行器 |
| **智能家居** | 家用设备联网、场景联动 | 智能门锁、智能家电、环境传感器 |
| **智慧门店** | 设备监控、数据分析 | 冷链设备、POS 机、监控摄像头 |
| **智慧园区** | 综合设施管理 | 能源系统、安防系统、停车系统 |

### 典型案例

- ✅ **多品牌 VRF 空调接入** - 支持 大金、日立、海尔、美的等品牌，统一数据标准
- ✅ **能耗管理系统** - 接入各类电表、水表、燃气表，实现能耗监控和分析
- ✅ **楼宇自控系统** - 集成 BACnet、Modbus 设备，实现统一控制和管理
- ✅ **工业设备联网** - 将传统工业设备连接到云平台，实现远程监控

---

## 🤝 参与贡献

欢迎参与 driver-box 的开发，您的贡献将帮助更多开发者！

### 贡献流程

1. **Fork** 本仓库
2. **创建特性分支** (`git checkout -b feat/AmazingFeature`)
3. **提交更改** (`git commit -m 'Add some AmazingFeature'`)
4. **推送到分支** (`git push origin feat/AmazingFeature`)
5. **提交 Pull Request**

### 代码规范

- 遵循 [Effective Go](https://go.dev/doc/effective_go) 编码规范
- 提交信息遵循 [Conventional Commits](https://www.conventionalcommits.org/)
- 添加必要的单元测试和文档注释
- 确保通过所有测试检查

---


---

## 📞 反馈与支持

### 获取帮助

- 📚 **[官方文档](https://ibuilding-x.github.io/driver-box/)** - 完整的使用文档和 API 参考
- 🐛 **[Issue 反馈](https://gitee.com/ibuilding-X/driver-box/issues)** - 报告 Bug 或提交功能请求
- 💬 **[讨论区](https://gitee.com/ibuilding-X/driver-box/discussions)** - 交流使用经验和最佳实践
- 🔍 **[DeepWiki](https://deepwiki.com/ibuilding-X/driver-box)** - AI 驱动的知识库问答

### 联系方式

如有商业合作需求或技术支持，请通过 Issue 或 Discussion 联系我们。

---

## 🙏 致谢

感谢以下开源项目的支持：

- [EdgeX Foundry](https://www.edgexfoundry.org/) - 边缘计算框架的启发
- [Golang](https://golang.org/) - 强大的 Go 语言生态
- [Yuin/gopher-lua](https://github.com/yuin/gopher-lua) - Lua 解释器
- 所有贡献者的代码贡献和建议

---

<div align="center">

**如果这个项目对您有帮助，请给我们一个 ⭐️ Star！**

[⬆ 回到顶部](#driver-box)

</div>
