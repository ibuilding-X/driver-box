package gwexport

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ibuilding-x/driver-box/pkg/driverbox/helper/cmanager"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/ibuilding-x/driver-box/internal/core"
	"github.com/ibuilding-x/driver-box/internal/dto"
	"github.com/ibuilding-x/driver-box/internal/export/discover"
	"github.com/ibuilding-x/driver-box/pkg/driverbox/config"
	"github.com/ibuilding-x/driver-box/pkg/driverbox/event"
	"github.com/ibuilding-x/driver-box/pkg/driverbox/helper"
	"github.com/ibuilding-x/driver-box/pkg/driverbox/helper/utils"
	"github.com/ibuilding-x/driver-box/pkg/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/pkg/driverbox/restful"
	"go.uber.org/zap"
)

// WebSocketPath websocket 服务路径
const WebSocketPath = "/ws/gateway-export"

var (
	errRegistered   = errors.New("already registered") // 主网关已注册错误
	errSelfRegister = errors.New("self register")      // 不能自注册错误
	errGatewayKey   = errors.New("gateway key error")  // 网关 Key 错误
	errDeviceID     = errors.New("device id error")    // 设备 ID 错误
)

// websocketService websocket 服务（提供 websocket 所需的所有服务功能，并非仅仅启动 websocket 服务端）
type websocketService struct {
	upGrader        websocket.Upgrader
	mainGateway     string          // 主网关 Key
	mainGatewayConn *websocket.Conn // 主网关连接
	mu              sync.Mutex
}

// Start 启动 websocket 服务
func (wss *websocketService) Start() {
	// 启动 websocket 服务，复用框架 http 服务
	restful.HttpRouter.HandlerFunc(http.MethodGet, WebSocketPath, wss.handler)
}

// handler 处理连接
func (wss *websocketService) handler(w http.ResponseWriter, r *http.Request) {
	// 升级 websocket 连接
	conn, err := wss.upGrader.Upgrade(w, r, nil)
	if err != nil {
		helper.Logger.Error("gateway export ws upgrade error", zap.Error(err))
		return
	}
	defer conn.Close()

	helper.Logger.Info("gateway export ws connect success", zap.String("remoteAddress", conn.RemoteAddr().String()))

	// 处理 websocket 消息
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			helper.Logger.Error("gateway export read ws message error", zap.Error(err))
			break
		}

		if err = wss.handleMessage(conn, message); err != nil {
			helper.Logger.Error("gateway export handle ws message error", zap.Error(err))
		}
	}
	wss.mainGatewayConn = nil
	wss.mainGateway = ""
	// ws 连接关闭
	helper.Logger.Warn("gateway export ws close", zap.String("remoteAddress", conn.RemoteAddr().String()))
}

// handleMessage 处理消息
func (wss *websocketService) handleMessage(conn *websocket.Conn, message []byte) error {
	var payload dto.WSPayload
	if err := json.Unmarshal(message, &payload); err != nil {
		return err
	}

	// 响应结构体
	var res dto.WSPayload

	switch payload.Type {
	case dto.WSForRegister: // 网关注册
		res.Type = dto.WSForRegisterRes
		if err := wss.gatewayRegister(conn, payload); err != nil {
			// 网关注册失败
			res.Error = err.Error()
		} else {
			// 返回子网关唯一标识
			res.GatewayKey = core.Metadata.SerialNo
			defer wss.sync()
		}
	case dto.WSForPing: // 心跳
		res.Type = dto.WSForPong
	case dto.WSForControl: // 控制指令
		res.Type = dto.WSForControlRes
		if err := wss.control(payload); err != nil {
			res.Error = err.Error()
		}
	case dto.WSForUnregister: // 网关注销
		res.Type = dto.WSForUnregisterRes
		if err := wss.gatewayUnregister(conn, payload); err != nil {
			res.Error = err.Error()
		}
	case dto.WSForReportRes: // 设备数据上报响应
		if payload.Error != "" {
			helper.Logger.Error("gateway export report error", zap.String("error", payload.Error))
		} else {
			helper.Logger.Info("gateway export report success")
		}
	case dto.WSForSyncModelsRes: // 模型数据同步响应
		if payload.Error != "" {
			helper.Logger.Error("gateway export sync models error", zap.String("error", payload.Error))
		} else {
			helper.Logger.Info("gateway export sync models success")
		}
	case dto.WSForSyncDevicesRes: // 设备数据同步响应
		if payload.Error != "" {
			helper.Logger.Error("gateway export sync devices error", zap.String("error", payload.Error))
		} else {
			helper.Logger.Info("gateway export sync devices success")
		}
	case dto.WSForSyncShadowRes: // 设备影子数据同步响应
		if payload.Error != "" {
			helper.Logger.Error("gateway export sync shadow error", zap.String("error", payload.Error))
		} else {
			helper.Logger.Info("gateway export sync shadow success")
		}
	default:
		return nil
	}

	// 发送响应消息
	if res.Type != 0 {
		if err := conn.WriteJSON(res); err != nil {
			return err
		}
	}

	return nil
}

// sync 同步数据（模型数据、设备列表、设备点位）
func (wss *websocketService) sync() {
	wss.syncModels()
	wss.syncDevices()
	wss.syncDevicesPoints()
}

// syncModels 同步设备模型数据
func (wss *websocketService) syncModels() {
	// 获取所有模型名称
	models := helper.CoreCache.Models()
	if len(models) == 0 {
		return
	}

	// 获取设备模型
	var deviceModels []config.DeviceModel
	for _, model := range models {
		deviceModel, ok := cmanager.GetModel(model.Name)
		if !ok {
			continue
		}

		// fix：同步模型时，应去除模型下设备
		deviceModel.Devices = nil

		// 修改模型名称，防止与主网关模型名称重复
		deviceModel.Name = wss.genGatewayModelName(deviceModel.Name)
		deviceModels = append(deviceModels, deviceModel)
	}

	// 发送模型数据
	var sendData dto.WSPayload
	sendData.Type = dto.WSForSyncModels
	sendData.Models = deviceModels

	if err := wss.sendJSONToWebSocket(sendData); err != nil {
		helper.Logger.Error("gateway export sync models error", zap.Error(err))
	}
}

// syncDevices 同步设备数据
func (wss *websocketService) syncDevices() {
	devices := helper.CoreCache.Devices()
	if len(devices) == 0 {
		return
	}

	// 优化设备数据
	// 1. 替换连接 Key 为模型名称，解决模型名称序列化丢失问题；
	// 2. 精简数据，移除不必要的字段，减少传输数据量；
	for i, device := range devices {
		devices[i] = config.Device{
			ID:            wss.genGatewayDeviceID(device.ID),
			Description:   device.Description,
			ConnectionKey: wss.genGatewayModelName(device.ModelName),
			Tags:          device.Tags,
			Properties:    device.Properties,
		}
	}

	// 发送设备数据
	var sendData dto.WSPayload
	sendData.Type = dto.WSForSyncDevices
	sendData.Devices = devices

	if err := wss.sendJSONToWebSocket(sendData); err != nil {
		helper.Logger.Error("gateway export sync devices error", zap.Error(err))
	}
}

// syncDevicesPoints 同步设备点位数据
func (wss *websocketService) syncDevicesPoints() {
	devices := helper.DeviceShadow.GetDevices()
	// 修改设备 ID
	for i, _ := range devices {
		devices[i].ID = wss.genGatewayDeviceID(devices[i].ID)
		devices[i].ModelName = wss.genGatewayModelName(devices[i].ModelName)
	}

	// 发送设备影子数据
	var sendData dto.WSPayload
	sendData.Type = dto.WSForSyncShadow
	sendData.Shadow = devices

	if err := wss.sendJSONToWebSocket(sendData); err != nil {
		helper.Logger.Error("gateway export sync shadow error", zap.Error(err))
	}
}

// sendDeviceData 发送设备数据（包含点位、事件等）
func (wss *websocketService) sendDeviceData(data plugin.DeviceData) {
	if data.ID == "" {
		return
	}

	if len(data.Values) == 0 && len(data.Events) == 0 {
		return
	}

	// 修改设备 ID
	data.ID = wss.genGatewayDeviceID(data.ID)

	// 修改事件数据
	if len(data.Events) > 0 {
		var events []event.Data
		for _, e := range data.Events {
			switch e.Code {
			case event.EventCodeDeviceStatus: // 设备状态
				// todo 事件定义暂时无法获取设备 ID
			case event.EventDeviceDiscover: // 设备发现
				var deviceDiscover discover.DeviceDiscover
				if err := utils.Conv2Struct(e.Value, &deviceDiscover); err == nil {
					deviceDiscover.ProtocolName = "driverbox"                                   // 修改协议名称
					deviceDiscover.ConnectionKey = wss.mainGateway                              // 修改连接 Key
					deviceDiscover.Device.ID = wss.genGatewayDeviceID(deviceDiscover.Device.ID) // 修改设备 ID
					deviceDiscover.Device.ConnectionKey = wss.mainGateway                       // 修改设备连接 Key
					deviceDiscover.Device.ModelName = wss.genGatewayModelName(deviceDiscover.Device.ModelName)
					events = append(events, event.Data{
						Code:  e.Code,
						Value: deviceDiscover,
					})
				}
			}
		}
		data.Events = events
	}

	// 汇总数据
	var sendData dto.WSPayload
	sendData.Type = dto.WSForReport
	sendData.DeviceData = data

	if err := wss.sendJSONToWebSocket(sendData); err != nil {
		helper.Logger.Error("gateway export send device data error", zap.Error(err))
	}
}

// gatewayRegister 处理网关注册
// 提示：主动释放 ws 连接逻辑放在客户端处理，服务端暂时不做处理
func (wss *websocketService) gatewayRegister(conn *websocket.Conn, payload dto.WSPayload) error {
	if payload.GatewayKey == "" {
		return errGatewayKey
	}

	wss.mu.Lock()
	defer wss.mu.Unlock()

	// 自注册
	if payload.GatewayKey == core.Metadata.SerialNo {
		return errSelfRegister
	}

	// 已注册
	if wss.mainGateway != "" && wss.mainGateway != payload.GatewayKey {
		return errRegistered
	}

	// 记录主网关信息
	wss.mainGateway = payload.GatewayKey
	wss.mainGatewayConn = conn

	return nil
}

// control 处理控制指令
func (wss *websocketService) control(payload dto.WSPayload) error {
	if payload.DeviceData.ID == "" {
		return errDeviceID
	}

	if pointNum := len(payload.DeviceData.Values); pointNum > 0 {
		// 解析设备 ID
		payload.DeviceData.ID = wss.parseGatewayDeviceID(payload.DeviceData.ID)
		return core.SendBatchWrite(payload.DeviceData.ID, payload.DeviceData.Values)
	}

	return nil
}

// gatewayUnregister 处理网关注销
func (wss *websocketService) gatewayUnregister(_ *websocket.Conn, payload dto.WSPayload) error {
	if payload.GatewayKey == "" {
		return errGatewayKey
	}

	wss.mu.Lock()
	defer wss.mu.Unlock()

	// 取消注册
	if wss.mainGateway != "" && wss.mainGateway == payload.GatewayKey {
		wss.mainGateway = ""
		wss.mainGatewayConn = nil
	}

	return nil
}

// genDeviceID 生成网关设备 ID
func (wss *websocketService) genGatewayDeviceID(id string) string {
	return fmt.Sprintf("%s/%s", core.Metadata.SerialNo, id)
}

// genGatewayModelName 生成网关模型名称（与主网关模型名称不能重复）
func (wss *websocketService) genGatewayModelName(name string) string {
	return fmt.Sprintf("%s_%s", core.Metadata.SerialNo, name)
}

// parseGatewayDeviceID 解析网关设备 ID
func (wss *websocketService) parseGatewayDeviceID(id string) string {
	return strings.ReplaceAll(id, core.Metadata.SerialNo+"/", "")
}

// sendJSONToWebSocket 发送 JSON 数据到 websocket
func (wss *websocketService) sendJSONToWebSocket(v interface{}) error {
	if wss.mainGateway == "" || wss.mainGatewayConn == nil {
		return nil
	}
	wss.mu.Lock()
	defer wss.mu.Unlock()
	return wss.mainGatewayConn.WriteJSON(v)
}
