---
title: Modbus插件
description: A reference page in my new Starlight docs site.
---

## 插件介绍

modbus插件用于连接modbus设备，支持 rtu、tcp、udp、tcp+tls 等通讯方式。

## modbus连接配置

| 配置项           | 必填 | 类型     | 参数说明                                                                                                          |
|---------------|----|--------|---------------------------------------------------------------------------------------------------------------|
| address       | 必填 | string | 连接地址：例如：`127.0.0.1:502` 、`/dev/ttyUSB0`                                                                       |
| mode          | 必填 | string | 通讯方式，支持类型：<ul><li>tcp</li><li>rtu</li><li>rtuovertcp</li><li>rtuoverudp</li><li>tcp+tls</li><li>udp</li></ul> |
| baudRate      | 选填 | int    | 波特率，仅当 mode 为`rtu`时需要设置。<br/>默认：`19200`                                                                       |
| dataBits      | 选填 | int    | 数据位 ，仅当 mode 为`rtu`时需要设置。<br/>默认：`8`                                                                          |
| stopBits      | 选填 | int    | 停止位 ，仅当 mode 为`rtu`时需要设置。<br/>有效范围：<ul><li>0</li><li>1</li><li>2</li></ul>                                    |
| parity        | 选填 | int    | 奇偶性校验 ，仅当 mode 为`rtu`时需要设置 。<br/>有效范围：<ul><li>0:None </li><li>1:EVEN</li><li>2:ODD  </li></ul>                |
| duration      | 选填 | string | 当前连接采集任务的执行周期。<br/>默认：`1s`。例如：`1s`、`1m`、`1h`、`1d`                                                             |
| batchReadLen  | 选填 | int    | 支持连续读的字节数。                                                                                                    |   
| batchWriteLen | 选填 | int    | 支持连续写的字节数。                                                                                                    |   
| retry         | 选填 | int    | 执行写操作出现失败时的重试次数，默认：3                                                                                          |
| virtual       | 选填 | bool   | 是否启用虚拟模式，默认：false                                                                                             |

### 示例

#### TCP模式

```json
<!--config.json-->
{
  ...
  "connections": {
    "192.168.16.111:502": {
      "address": "192.168.16.111:502",
      "batchReadLen": 50,
      "batchWriteLen": 10,
      "enable": true,
      "minInterval": 500,
      "mode": "tcp",
      "timeout": 5000,
      "virtual": false
    }
  },
  "protocolName": "modbus"
}

```

#### RTU模式

```json
<!--config.json-->
{
  ...
  "connections": {
    "/dev/ttyS5": {
      "address": "/dev/ttyS5",
      "batchReadLen": 10,
      "baudRate": 9600,
      "dataBits": 8,
      "enable": true,
      "minInterval": 100,
      "mode": "rtu",
      "parity": 0,
      "retry": 0,
      "stopBits": 1,
      "timeout": 2000,
      "virtual": false
    }
  },
  "protocolName": "modbus"
}
```

## 扩展点表配置

| 配置项          | 必填 | 类型     | 参数说明                                                                                                     |
|--------------|----|--------|----------------------------------------------------------------------------------------------------------|
| primaryTable | 必填 | string | 寄存器类型，支持类型：<ul><li>HOLDING_REGISTER</li><li>COIL</li><li>DISCRETE_INPUT</li><li>INPUT_REGISTER</li></ul> |
| startAddress | 必填 | string | 点位所在的寄存器地址，详见：[startAddress配置说明](#startAddress配置说明)                                                      |
| rawType      | 必填 | string | 数据原始类型 ，详见：[rawType配置说明](#rawType配置说明)                                                                   |
|duration|选填|string| 当前点位值的采集频率。<br/>默认：`1s`。例如：`1s`、`1m`、`1h`、`1d`                                                           |
| wordSwap     | 选填 | int    |                                                                                                          |
| byteSwap     | 选填 | int    |                                                                                                          |
| bit | 选填 | string | 点位值所处的比特位起始位置，默认：0                                                                                       |
| bitLen | 选填 | int | 比特位长度                                                                                                    |

### startAddress配置说明

寄存器地址存在三种表达方式：

1. 十六进制表示  
   使用前缀"0x"来指示该地址是十六进制形式，例如："0x0400" 表示第1024个寄存器地址。
   解析时，会去掉"0x"并将其转换为十进制。
2. 十进制表示   
   使用后缀"d"来区分十进制地址，例如："40001d" 表示第40001个寄存器地址。
   在解析时，会移除"d"并直接转换为十进制的uint16值。
3. 特定格式表示   
   对于长度为5位的数字字符串，遵循特定的地址映射规则：
    - 地址范围在1-9999，实际地址 = 字符串值 - 1。
    - 地址范围在10000-19999，实际地址 = 字符串值 - 10001。
    - 地址范围在30000-39999，实际地址 = 字符串值 - 30001。
    - 地址范围在40000-49999，实际地址 = 字符串值 - 40001。
    - 其他值被视为无效。

请注意，确保提供的地址符合上述格式，否则函数可能会返回错误。在与Modbus兼容的设备通信时，正确地表示和转换寄存器地址是至关重要的。

:::tip
具体算法可参见`internal/plugins/modbus/connector.go#castModbusAddress`
:::

### rawType配置说明

rawType 表示从寄存器读取到的字节数值代表的真实数据类型。

当寄存器类型为 `COIL`、`DISCRETE_INPUT` 时，rawType 参数无效。

rawType 适用于寄存器类型为：`INPUT_REGISTER`、`HOLDING_REGISTER` 的场景，且不同数据类型对应的寄存器字节数对照关系如下：

|         | 1字节     |2字节|4字节|
|---------|---------|-|--|
| int16   | &#10004; |||          
| uint16  | &#10004; |||          
| int32   |         |&#10004;||          
| uint32  |         |&#10004;||          
| float32 |         |&#10004;||          
| int64   |         ||&#10004;|          
| uint64  |         ||&#10004;|          
| float64 |         ||&#10004;|          

## 性能优化

## 虚拟设备