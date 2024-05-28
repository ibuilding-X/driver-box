---
title: 项目简介
description: A guide in my new Starlight docs site.
sidebar:
  order: 1
---

## 1. 简介
DriverBox 是一款支持泛化协议接入的边缘网关框架， 以插件化的形式融合了 Modbus、Bacnet、HTTP、MQTT 等主流协议，同时也支持基于TCP的各类私有化协议对接。

![](/framework.svg)

我们期望 DriverBox 能够为相关人士提供更加高效、舒适的设备接入体验。 通过对各类设备的通信协议和数据交互形式进行抽象，定义了一套标准流程以涵盖泛化协议的共性处理逻辑，并结合动态解析脚本（Lua）填补其中的差异化部分。

以此解决设备接入过程中存在的驱动工程数量爆炸；接入标准难以规范化等问题。



## 名词解释

### 虚拟设备
虚拟设备是 DriverBox 框架提供的一种设备通讯方式模拟能力。
使用户可在无需对接真实设备的情况下，进行本地设备配置和调试。

开启虚拟设备模式，只需将 connections 中的 `virtual` 配置项设置为 `true`。

现以支持以下几种通讯插件：
- Bacnet
- [Modbus](/plugins/modbus/#虚拟设备)