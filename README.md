# DriverBox

DriverBox 是一款基于物联网开源框架 Edgex（opens new window）打造的泛化协议接入服务。 以插件化的形式融合了 Modbus、TCP、HTTP、MQTT 等主流协议，同时也支持基于TCP的各类私有化协议对接。

我们期望 DriverBox 能够为相关人士提供更加高效、舒适的设备接入体验。 通过对各类设备的通信协议和数据交互形式进行抽象，定义了一套标准流程以涵盖泛化协议的共性处理逻辑，并结合动态解析脚本（Lua、Javascript、Python）填补其中的差异化部分。


## 文档

[快速开始](https://ibuilding-x.gitee.io/driver-box/quick_start/)


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

3. 启动 EdgeX 环境

```bash
# 默认提供的 docker-compose.yml 采用的是 openyurt 多架构镜像
docker compose up -d

# 可通过以下命令查看服务状态
docker compose ps -a
```

## 本地运行

1. 修改 main.go 文件

```go
func main() {
  _ = os.Setenv("EDGEX_SECURITY_SECRET_STORE", "false")

  // 正式环境需注释掉
  localMode("127.0.0.1", "59999", "127.0.0.1") // 按照实际情况修改

  sd := driver.Driver{}
  startup.Bootstrap(serviceName, version, &sd)
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