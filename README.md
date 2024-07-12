# DriverBox

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