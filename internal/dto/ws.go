package dto

import (
	"github.com/ibuilding-x/driver-box/driverbox/event"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
)

type WSPayloadType int8

const (
	WSForRegister      WSPayloadType = iota + 1 // 注册请求
	WSForRegisterRes                            // 注册响应
	WSForUnregister                             // 取消注册请求
	WSForUnregisterRes                          // 取消注册成功响应
	WSForPing                                   // 心跳
	WSForPong                                   // 心跳响应
	WSForReport                                 // 上报请求
	WSForReportRes                              // 上报响应
	WSForControl                                // 控制请求
	WSForControlRes                             // 控制响应
)

// WSPayload websocket 消息体
type WSPayload struct {
	Type       WSPayloadType      `json:"type"`        // 消息类型
	GatewayKey string             `json:"gateway_key"` // 网关唯一标识，当 type 为 WSForConnect、 WSForDisconnect 时，此字段必填
	DeviceID   string             `json:"device_id"`   // 设备ID，当 type 为 WSForReport、 WSForControl 时，此字段必填
	Points     []plugin.PointData `json:"points"`      // 控制点数据，当 type 为 WSForReport、 WSForControl 时，此字段必填
	Events     []event.Data       `json:"events"`      // 事件数据，当 type 为 WSForReport、 WSForControl 时，此字段必填
	Error      string             `json:"error"`       // 错误信息，当 type 为 WSForRegisterRes、 WSForUnregisterRes、 WSForControlRes 时，此字段必填
}
