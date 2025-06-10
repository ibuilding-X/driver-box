# DriverBox
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/ibuilding-X/driver-box)

## 文档

[快速开始](https://ibuilding-x.github.io/driver-box/)

## 简介
driver-box 是一款支持泛化协议接入的边缘网关框架， 以插件化的形式融合了 Modbus、Bacnet、HTTP、MQTT 等主流协议，同时也支持基于TCP的各类私有化协议对接。
![](https://ibuilding-x.github.io/driver-box/framework.svg)

我们期望 driver-box 能够为相关人士提供更加高效、舒适的设备接入体验。

通过对各类设备的通信协议和数据交互形式进行抽象，定义了一套标准流程以涵盖各类通信协议的共性逻辑，并结合动态解析脚本（Lua）填补其中的差异化部分。

以此解决设备接入过程中存在的驱动工程数量爆炸；接入标准难以规范化等问题。

## 特性
### 免费开源
采用商业友好的 Apache-2.0 开源协议，使其成为 IoT 生态圈的极佳选择。

### 架构
以 Golang 为主要开发语言，可编译出适配 amd64、arm64、armv7、x86 等系统架构的可执行程序。 存储空间和运行内存控制在十几MB，满足低规格网关的运行需求。

采用高度统一的配置化方式对接各类通讯设备。理想情况下只需编写一个 JSON 文件便可完成设备接入，亦可结合 lua 脚本实现复杂设备的数据加工。

通过精心的架构设计，三方用户可无限扩展边缘网关的设备通讯能力和应用服务能力。

### API
driver-box 没有提供配套的 UI 界面，但开放了大量实用 RestAPI。用户可以自由设计网关 UI，定制出极致用户体验的边缘产品。

### 应用场景
driver-box 适用于多种场景，包括智能家居、智慧楼宇、智慧工厂、智慧门店。它促进了设备数据的采集与场景融合，实现了万物皆可连、万物皆可互联、万物皆可智联。

## 安装

1. 下载源代码

```bash
git clone https://gitee.com/iBUILDING-X/driver-box.git
```

2. 加载 go 依赖

```bash
cd driver-box
go mod vendor # 国内用户可以切换源：go env -w GOPROXY=https://goproxy.cn,direct
```

## 本地运行

1. 打开 main.go 文件

```go
func main() {
    driverbox.Start([]export.Export{&export.DefaultExport{}})
    select {}
}
```

2. 启动 driver-box

```bash
go run main.go
```

## 参与贡献

1.  Fork 本仓库
2.  新建 Feat_xxx 分支
3.  提交代码
4.  新建 Pull Request

## 反馈

如果您有任何问题，请通过 [issues](https://gitee.com/iBUILDING-X/driver-box/issues) 快速反馈

## 致谢

- [EdgeX Foundry](https://www.edgexfoundry.org/)
