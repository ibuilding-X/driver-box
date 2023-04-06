package helper

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/edgexfoundry/device-sdk-go/v2/pkg/service"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/clients/http"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/clients/interfaces"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/dtos"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/dtos/requests"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/models"
	"time"
)

var noticeClient interfaces.NotificationClient
var noticeLabels = []string{"Default"} // 通知固定 Labels
var noticeCategory = "normal"          // 通知固定 Category

type EventType string
type DeviceEventEnum string

const (
	CommandAck      EventType = "CommandAck"      // 指令相应
	ScheduleTrigger EventType = "ScheduleTrigger" // 时间表触发
	DeviceEvent     EventType = "DeviceEvent"     // 设别事件
)

const (
	DeviceEventOnline  DeviceEventEnum = "Online"  // 设备在线事件
	DeviceEventOffline DeviceEventEnum = "Offline" // 设备离线事件
)

// EventModel 自定义事件模型
type EventModel struct {
	EventType       EventType   `json:"eventType"`       // 事件类型（指令相应、时间表触发、设备事件）
	ReportTimestamp int64       `json:"reportTimestamp"` // 上报时间戳（毫秒级）
	EventData       interface{} `json:"eventData"`       // 事件数据
	Description     string      `json:"description"`     // 事件描述
}

// DeviceEventModel 设备事件模型
type DeviceEventModel struct {
	DeviceSN string                 `json:"deviceSN"` // 设备SN
	Type     DeviceEventEnum        `json:"type"`     // 事件名称
	Data     map[string]interface{} `json:"data"`     // 事件数据
}

// InitNotification 初始化 Notification Client
func InitNotification() {
	url := fmt.Sprintf("http://%s:%d", "edgex-support-notifications", 59860)
	noticeClient = http.NewNotificationClient(url)
}

// SendStatusChangeNotification 发送设备状态变更通知
func SendStatusChangeNotification(deviceName string, online bool) error {
	var reqs []requests.AddNotificationRequest
	noticeData := newStatusChangeNoticeData(deviceName, online)
	notification := dtos.NewNotification(noticeLabels, noticeCategory, noticeData, service.RunningService().ServiceName, models.Normal)
	req := requests.NewAddNotificationRequest(notification)
	reqs = append(reqs, req)

	_, err := noticeClient.SendNotification(context.Background(), reqs)
	return err
}

// newStatusChangeNoticeData 实例化状态变更通知数据
func newStatusChangeNoticeData(deviceName string, online bool) string {
	var status DeviceEventEnum
	if online {
		status = DeviceEventOnline
	} else {
		status = DeviceEventOffline
	}
	data := EventModel{
		EventType:       DeviceEvent,
		ReportTimestamp: time.Now().UnixMilli(),
		EventData: DeviceEventModel{
			DeviceSN: deviceName,
			Type:     status,
		},
	}
	b, _ := json.Marshal(data)
	return string(b)
}
