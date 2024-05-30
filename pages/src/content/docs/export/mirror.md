---
title: 镜像设备
description: A guide in my new Starlight docs site.
---

当设备接入通过 driver-box 接入后，会在内存中基于设备的 **物模型** 生成一个对象实例，
上层应用可通过该实例进行设备状态的获取以及下行指令的发送。 

**镜像设备，则是在该通讯设备的对象实例基础上，构造出了一个拥有同等读写能力的虚拟设备。**

![](/driver-box/mirror_0.svg)

## 应用场景
**场景一**    
假如我们需要为客户打造一款标准的智慧化解决方案，但是同一类别的设备，不同客户采购的品牌可能各不相同。

即便对用户而言，这些设备有着相似，甚至相同的开关、模式等控制能力。
但在通讯方面，以及物模型的定义上，它们却存在着巨大差异。

此时，要么边缘网关只管设备接入，产品体验的一致性由上层应用软件不停的适配各厂家。

又或者，**通过 driver-box 镜像设备能力，在设备接入时便完成通用模型的适配。如此一来，上层应用无需为设备的兼容性不断发布新版本。**
![](/driver-box/mirror_1.svg)

**场景二**     
在某些复杂场景下，实施人员会将设备按通讯点位的方式接入。
至于如何以物模型的结构来定义这个场景，可能是由其他角色的人来负责。

此时，这个负责物模型标准化落地的人，可能面临的情况是物模型中定义的点，来源于不同种类的通讯方式；或者同一种通讯方式的不同链路。

这种工况会对智能化建设的进程带来些许阻碍，而采用 driver-box 镜像设备能力，可以比较容易将该痛点消除在设备接入环节。

![](/driver-box/mirror_2.svg)

## 使用方式

假定我们通过某种通讯插件已经完成了照明设备的接入，物模型中仅一个`onOff`点位。接入的设备ID分别为：
- switch-1
- switch-2

### 第一步：配置文件
接下来，通过镜像设备能力，将这两个设备的开关点位，汇集成一个新的设备实例，只需要进行如下配置：

```json title=config.json {9,13-14,,18,22-23,28,37}
{
  "deviceModels": [
    {
      "name": "swtich",
      "description": "开关",
      "devicePoints": [
        {
          "description": "开关",
          "name": "onOff-1",
          "readWrite": "RW",
          "reportMode": "change",
          "valueType": "int",
          "rawDevice": "swtich-1",
          "rawPoint": "onOff"
        },
        {
          "description": "开关",
          "name": "onOff-2",
          "readWrite": "RW",
          "reportMode": "change",
          "valueType": "int",
          "rawDevice": "swtich-2",
          "rawPoint": "onOff"
        }
      ],
      "devices": [
        {
          "id": "mirror-swtich-3",
          "description": "1号开关",
          "ttl": "5m"
        }
      ]
    }
  ],
  "connections": {
  },
  "protocolName": "mirror"
}
```
1. 镜像设备在原设备点表配置的基础上，增加了两个配置项：
    - rawDevice：原设备ID
    - rawPoint：原设备点位名称
2. 镜像设备不存在连接信息，因此无需配置 `connections`。
3. 镜像设备的协议名称为：`mirror`。

### 第二步：启用服务
镜像服务已内置在 driver-box 框架中，只需在启动时将 `mirror.NewExport()` 注入即可。
```go ins="mirror.NewExport()"
func main() {
	driverbox.Start([]export.Export{mirror.NewExport()})
}
```