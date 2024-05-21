---
title: Modbus插件
description: A reference page in my new Starlight docs site.
---

## modbus连接配置
| 配置项      | 必填 | 类型     | 参数说明                                                            |
|----------|----|--------|-----------------------------------------------------------------|
| address  | 必填 | string | 连接地址：例如：`127.0.0.1:502` 、`/dev/ttyUSB0`                         |
| mode     | 必填 | string | 通讯方式，支持类型：`rtu`、`rtuovertcp`、`rtuoverudp`、`tcp`、`tcp+tls`、`udp` |
| baudRate | 选填 | int    | 波特率，仅当 mode 为`rtu`时需要设置，默认：`19200`                              |
| dataBits | 选填 | int    | 数据位 ，仅当 mode 为`rtu`时需要设置，默认：`8`                                 |
| stopBits | 选填 | int    | 停止位 ，仅当 mode 为`rtu`时需要设置。有效范围：`0`、1`、`2`                        |
| parity   | 选填 | int    | 奇偶性校验 ，仅当 mode 为`rtu`时需要设置 。有效范围：0:None ,1:EVEN ,2:ODD          |
| duration | 选填 | string | 当前连接采集任务的执行周期，默认：`1s`。例如：`1s`、`1m`、`1h`、`1d`                    |
| maxLen   | 选填 | int    | 最长连续读个数。相邻点位间隔若低于maxLen，将会一次性读出，默认：`32`                         
| retry    | 选填 | int    | 执行写操作出现失败时的重试次数，默认：3                                            |

## 扩展点表配置
