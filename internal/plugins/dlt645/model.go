package dlt645

import (
	"time"

	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
)

type primaryTable string

const BatchReadMode plugin.EncodeMode = "batchRead"

// ConnectionConfig 连接器配置
type ConnectionConfig struct {
	plugin.BaseConnection
	Address            string `json:"address"`            // 地址：例如：127.0.0.1:502
	BaudRate           uint   `json:"baudRate"`           // 波特率（仅串口模式）
	DataBits           uint   `json:"dataBits"`           // 数据位（仅串口模式）
	StopBits           uint   `json:"stopBits"`           // 停止位（仅串口模式）
	Parity             string `json:"parity"`             // 奇偶性校验（仅串口模式）
	MinInterval        uint16 `json:"minInterval"`        // 最小读取间隔
	Timeout            uint16 `json:"timeout"`            // 请求超时
	Retry              int    `json:"retry"`              // 重试次数
	AutoReconnect      bool   `json:"autoReconnect"`      //自动重连
	ProtocolLogEnabled bool   `json:"protocolLogEnabled"` // 协议解析日志
}

// Point 点位
type Point struct {
	config.Point
	//冗余设备相关信息
	DeviceId string

	//点位采集周期
	Duration  string `json:"duration"`
	Address   uint16
	Quantity  uint16 `json:"quantity"`
	DataMaker string `json:"dataMaker"`
}

// 采集组
type slaveDevice struct {
	// 通讯设备，采集点位可以对应多个物模型设备
	address string
	//分组
	pointGroup []*pointGroup
}

type pointGroup struct {
	index      int           //分组索引
	Duration   time.Duration //采集间隔
	LatestTime time.Time     //上一次采集时间
	Address    string        //起始地址
	Quantity   uint16        //数量
	Points     []*Point
	DataMaker  string // dlt645标准中点位标识
	SlaveId    string // 电表地址
}

// Connector#Send接入入参
type command struct {
	Mode  plugin.EncodeMode // 模式
	Value interface{}
}
