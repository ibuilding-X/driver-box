package dto

import (
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/event"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
)

type WSPayloadType int8

const (
	WSForRegister       WSPayloadType = iota + 1 // 注册请求
	WSForRegisterRes                             // 注册响应
	WSForUnregister                              // 取消注册请求
	WSForUnregisterRes                           // 取消注册成功响应
	WSForPing                                    // 心跳
	WSForPong                                    // 心跳响应
	WSForReport                                  // 上报请求
	WSForReportRes                               // 上报响应
	WSForControl                                 // 控制请求
	WSForControlRes                              // 控制响应
	WSForSyncModels                              // 同步模型请求
	WSForSyncModelsRes                           // 同步模型响应
	WSForSyncDevices                             // 同步设备请求
	WSForSyncDevicesRes                          // 同步设备响应
)

// WSPayload websocket 消息体
type WSPayload struct {
	Type       WSPayloadType      `json:"type"`        // 消息类型
	GatewayKey string             `json:"gateway_key"` // 网关唯一标识，当 type 为 WSForConnect、 WSForDisconnect 时，此字段必填
	DeviceID   string             `json:"device_id"`   // 设备ID，当 type 为 WSForReport、 WSForControl 时，此字段必填
	Points     []plugin.PointData `json:"points"`      // 控制点数据，当 type 为 WSForReport、 WSForControl 时，此字段必填
	Events     []event.Data       `json:"events"`      // 事件数据，当 type 为 WSForReport、 WSForControl 时，此字段必填
	Models     []config.Model     `json:"models"`      // 模型数据，当 type 为 WSForSyncModels 时，此字段必填
	Devices    []config.Device    `json:"devices"`     // 设备数据，当 type 为 WSForSyncDevices 时，此字段必填
	Error      string             `json:"error"`       // 错误信息，当 type 为 WSForRegisterRes、 WSForUnregisterRes、 WSForControlRes 时，此字段必填
}
