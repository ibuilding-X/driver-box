package gwexport

import (
	"encoding/json"
	"errors"
	"github.com/gorilla/websocket"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/internal/dto"
	"go.uber.org/zap"
	"net/http"
	"sync"
)

// WebSocketPath websocket 服务路径
const WebSocketPath = "/ws/gateway-export"

var (
	errRegistered = errors.New("already registered") // 已注册错误
	errGatewayKey = errors.New("gateway key error")  // 网关 Key 错误
)

// websocketService websocket 服务（提供 websocket 所需的所有服务功能，并非仅仅启动 websocket 服务端）
type websocketService struct {
	upGrader    websocket.Upgrader
	mainGateway string // 主网关 Key
	mu          sync.Mutex
}

// Start 启动 websocket 服务
func (wss *websocketService) Start() {
	// 启动 websocket 服务，复用框架 http 服务
	http.HandleFunc(WebSocketPath, wss.handler)
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

	// ws 连接关闭
	helper.Logger.Warn("gateway export ws close", zap.String("addr", conn.RemoteAddr().String()))
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
		if err := wss.gatewayRegister(payload); err != nil {
			res.Error = err.Error()
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
		if err := wss.gatewayUnregister(payload); err != nil {
			res.Error = err.Error()
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

// syncModels 同步设备模型数据
func (wss *websocketService) syncModels() {
	// todo something
}

// syncDevices 同步设备数据
func (wss *websocketService) syncDevices() {
	// todo something
}

// sendDeviceData 发送设备数据（包含点位、事件等）
func (wss *websocketService) sendDeviceData(data plugin.DeviceData) {
	// todo something
}

// gatewayRegister 处理网关注册
// 提示：主动释放 ws 连接逻辑放在客户端处理，服务端暂时不做处理
func (wss *websocketService) gatewayRegister(payload dto.WSPayload) error {
	if payload.GatewayKey == "" {
		return errGatewayKey
	}

	wss.mu.Lock()
	defer wss.mu.Unlock()

	// 已注册
	if wss.mainGateway != "" && wss.mainGateway != payload.GatewayKey {
		return errRegistered
	}

	// 记录主网关 Key
	wss.mainGateway = payload.GatewayKey

	return nil
}

func (wss *websocketService) control(payload dto.WSPayload) error {
	// todo something

	return nil
}

// gatewayUnregister 处理网关注销
func (wss *websocketService) gatewayUnregister(payload dto.WSPayload) error {
	if payload.GatewayKey == "" {
		return errGatewayKey
	}

	wss.mu.Lock()
	defer wss.mu.Unlock()

	// 取消注册
	if wss.mainGateway != "" && wss.mainGateway == payload.GatewayKey {
		wss.mainGateway = ""
	}

	return nil
}
