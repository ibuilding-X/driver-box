# DriverBox

## Document

[Quick start](https://ibuilding-x.github.io/driver-box/)
 

## Install

1. Download The Source Code

```bash
git clone https://gitee.com/iBUILDING-X/driver-box.git
```

2. Load GO dependencies

```bash
cd driver-box
go mod vendor # 国内用户可以切换源：go env -w GOPROXY=https://goproxy.cn,direct
```

## Run locally

1. Open the main.go file

```go
func main() {
    driverbox.Start([]export.Export{&export.DefaultExport{}})
    select {}
}
```

2. Start the driver box

```bash
go run main.go
```

## Participate and contribute

1. Fork's own warehouse
2. Create a new Feat_xxx branch
3. Submit code
4. Create a new Pull Request

## Feedback

If you have any questions, please contact [issues](https://gitee.com/iBUILDING-X/driver-box/issues) Quick feedback

## Thank

- [EdgeX Foundry](https://www.edgexfoundry.org/)
