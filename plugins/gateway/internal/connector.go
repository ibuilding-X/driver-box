package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ibuilding-x/driver-box/driverbox"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/exports/gateway"
	"go.uber.org/zap"
)

var (
	ErrIP = errors.New("IP address not valid")
)

var (
	defaultTimeout           = 5 * time.Second  // 默认连接超时时间
	defaultReconnectInterval = 30 * time.Second // 默认重连间隔
)

// connectorConfig 连接配置
type connectorConfig struct {
	IP                string `json:"ip"`                 // 子网关 IP 地址
	Timeout           string `json:"timeout"`            // 连接超时时间
	ReconnectInterval string `json:"reconnect_interval"` // 重连间隔

	timeout           time.Duration // 存储连接超时时间
	reconnectInterval time.Duration // 存储重连间隔
}

// checkAndRepair 检查并修复配置
func (c *connectorConfig) checkAndRepair() error {
	// 检查 IP 地址是否合法
	if net.ParseIP(c.IP) == nil {
		return ErrIP
	}
	c.timeout = defaultTimeout
	c.reconnectInterval = defaultReconnectInterval
	// 检查超时时间
	if timeout, err := time.ParseDuration(c.Timeout); err == nil {
		c.timeout = timeout
	}
	// 检查重连间隔
	if reconnectInterval, err := time.ParseDuration(c.ReconnectInterval); err == nil {
		c.reconnectInterval = reconnectInterval
	}
	return nil
}

type encodeData struct {
	Mode plugin.EncodeMode
	plugin.DeviceData
}

type connector struct {
	conf      connectorConfig
	conn      *websocket.Conn // 存储 websocket 连接
	mu        sync.Mutex
	destroyed bool
}

func (c *connector) Encode(deviceId string, mode plugin.EncodeMode, values ...plugin.PointData) (res interface{}, err error) {
	return encodeData{
		Mode: mode,
		DeviceData: plugin.DeviceData{
			ID:     deviceId,
			Values: values,
		},
	}, nil
}

func (c *connector) Decode(raw interface{}) (res []plugin.DeviceData, err error) {
	payload, ok := raw.(gateway.WSPayload)
	if !ok {
		return nil, errors.New("invalid payload")
	}

	res = append(res, payload.DeviceData)

	return
}

func (c *connector) Send(data interface{}) (err error) {
	if c.conn == nil {
		return errors.New("not connected")
	}

	// 数据解析
	ed, ok := data.(encodeData)
	if !ok {
		return errors.New("data is not encodeData")
	}

	// 过滤指令，仅处理控制指令
	if ed.Mode != plugin.WriteMode {
		return nil
	}

	// 下发控制
	c.control(ed.DeviceData)
	return nil
}

func (c *connector) Release() (err error) {
	return nil
}

// connect 连接到子网关
// * 会阻塞进程，需携程处理
// * 需要实现重连机制
func (c *connector) connect() {
	url := fmt.Sprintf("ws://%s:%s%s", c.conf.IP, helper.EnvConfig.HttpListen, gateway.WebSocketPath)
	dialer := &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: c.conf.timeout,
	}

	for {
		helper.Logger.Info("gateway plugin connect to gateway", zap.String("url", url))

		// 建立连接
		var conn *websocket.Conn
		var err error

		if conn, _, err = dialer.Dial(url, nil); err == nil {
			// 连接成功
			helper.Logger.Info("gateway plugin connect to gateway success")
			// 存储连接
			c.conn = conn
			// 发送注册消息
			go c.register()
			// 接收子网关数据
			for {
				var message []byte
				if _, message, err = conn.ReadMessage(); err != nil {
					break
				}
				// 处理子网关数据
				if err = c.handleWebSocketMessage(conn, message); err != nil {
					helper.Logger.Error("gateway plugin handle websocket message failed", zap.Error(err), zap.Any("message", message))
				}
			}
		}

		// 连接失败或断开连接
		// 关闭连接
		if conn != nil {
			if err = conn.Close(); err != nil {
				helper.Logger.Error("gateway plugin close websocket connection failed", zap.Error(err))
			}
		}
		// 删除存储的 websocket 连接
		c.conn = nil
		// 已销毁连接不再重连
		if c.destroyed {
			helper.Logger.Info("gateway plugin destroyed, stop reconnect")
			break
		}
		// 延迟重连
		helper.Logger.Error("gateway plugin connect to gateway failed, retry after 30 seconds")
		time.Sleep(c.conf.reconnectInterval)
	}
}

// handleWebSocketMessage 处理子网关数据
func (c *connector) handleWebSocketMessage(conn *websocket.Conn, message []byte) (err error) {
	var payload gateway.WSPayload
	if err = json.Unmarshal(message, &payload); err != nil {
		return err
	}

	// 响应结构体
	var res gateway.WSPayload

	switch payload.Type {
	case gateway.WSForRegisterRes: // 注册响应
		return c.registerRes(payload)
	case gateway.WSForUnregisterRes: // 注销响应
		return c.unregisterRes(payload)
	case gateway.WSForPong: // 心跳响应
		return c.pong(payload)
	case gateway.WSForControlRes: // 控制响应
		return c.controlRes(payload)
	case gateway.WSForSyncModels: // 接收模型同步数据
		return c.syncModels(payload)
	case gateway.WSForSyncDevices: // 接收设备同步数据
		return c.syncDevices(payload)
	case gateway.WSForSyncShadow: // 接收影子同步数据
		return c.syncShadow(payload)
	case gateway.WSForReport: // 接收上报数据
		result, err := c.Decode(payload)
		if err != nil {
			return err
		}
		driverbox.Export(result)
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

// sendWebSocketPayload 向子网关发送数据
func (c *connector) sendWebSocketPayload(payload gateway.WSPayload) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		return c.conn.WriteJSON(payload)
	}

	return nil
}

// register 向子网关发送注册消息
func (c *connector) register() {
	// 延迟发送注册消息，防止无法收到注册响应消息
	time.Sleep(time.Second)

	if err := c.sendWebSocketPayload(gateway.WSPayload{
		Type:       gateway.WSForRegister,
		GatewayKey: c.conf.IP, // 网关唯一标识
	}); err != nil {
		helper.Logger.Error("gateway plugin send register payload failed", zap.String("IP", c.conf.IP), zap.Error(err))
	}
}

// registerRes 处理注册响应
func (c *connector) registerRes(payload gateway.WSPayload) error {
	if payload.Error != "" {
		helper.Logger.Error("gateway plugin register failed", zap.String("IP", c.conf.IP), zap.String("error", payload.Error))
		return nil
	}

	// 注册成功
	helper.Logger.Info("gateway plugin register success", zap.String("IP", c.conf.IP))
	// 更新网关设备状态
	return helper.DeviceShadow.SetDevicePoint(c.conf.IP, "SN", payload.GatewayKey)
}

// unregister 向子网关发送注销消息
func (c *connector) unregister() {
	// todo 暂时未使用
	if err := c.sendWebSocketPayload(gateway.WSPayload{
		Type:       gateway.WSForUnregister,
		GatewayKey: driverbox.GetMetadata().SerialNo, // 网关唯一标识
	}); err != nil {
		helper.Logger.Error("gateway plugin send unregister payload failed", zap.String("IP", c.conf.IP))
	}
}

// unregisterRes 处理注销响应
func (c *connector) unregisterRes(payload gateway.WSPayload) error {
	if payload.Error != "" {
		helper.Logger.Error("gateway plugin unregister failed", zap.String("IP", c.conf.IP), zap.String("error", payload.Error))
	}

	// 注销成功
	helper.Logger.Info("gateway plugin unregister success", zap.String("IP", c.conf.IP))
	// 更新网关设备状态
	return helper.DeviceShadow.SetOffline(c.conf.IP)
}

// ping 向子网关发送心跳消息
func (c *connector) ping() {
	if err := c.sendWebSocketPayload(gateway.WSPayload{
		Type: gateway.WSForPing,
	}); err != nil {
		helper.Logger.Error("gateway plugin send ping payload failed", zap.String("IP", c.conf.IP))
	}
}

// pong 处理心跳响应
func (c *connector) pong(_ gateway.WSPayload) error {
	helper.Logger.Debug("gateway plugin pong", zap.String("IP", c.conf.IP))
	return nil
}

func (c *connector) syncModels(payload gateway.WSPayload) error {
	if len(payload.Models) > 0 {
		var errCounter int
		for _, model := range payload.Models {
			// fix: 同步模型时，移除模型下所有设备
			model.Devices = nil

			err := driverbox.CoreCache().AddModel(ProtocolName, model)
			if err != nil {
				errCounter++
				helper.Logger.Error("gateway plugin add model failed", zap.Any("model", model), zap.Error(err))
			}
		}

		if errCounter > 0 {
			c.syncModelsRes(fmt.Errorf("sync models failed: models count: %d, error count: %d", len(payload.Models), errCounter))
		} else {
			c.syncModelsRes(nil)
		}
	}
	return nil
}

// syncModelsRes 发送模型同步响应
func (c *connector) syncModelsRes(err error) {
	var errMsg string
	if err != nil {
		errMsg = err.Error()
	}

	if err = c.sendWebSocketPayload(gateway.WSPayload{
		Type:  gateway.WSForSyncModelsRes,
		Error: errMsg,
	}); err != nil {
		helper.Logger.Error("gateway plugin send sync models res failed", zap.String("IP", c.conf.IP))
	}
}

func (c *connector) syncDevices(payload gateway.WSPayload) error {
	if len(payload.Devices) > 0 {
		var errCounter int

		for _, device := range payload.Devices {
			// 替换连接 key
			device.ModelName = device.ConnectionKey
			device.ConnectionKey = c.conf.IP

			// 获取本地设备信息
			if localDevice, ok := driverbox.CoreCache().GetDevice(device.ID); ok {
				// 优先使用本地设备信息
				device.Description = localDevice.Description
				device.Tags = localDevice.Tags
				device.Properties = localDevice.Properties
			}

			err := driverbox.CoreCache().AddOrUpdateDevice(device)
			if err != nil {
				errCounter++
				helper.Logger.Error("gateway plugin add device failed", zap.Any("device", device))
			}
		}

		if errCounter > 0 {
			c.syncDevicesRes(fmt.Errorf("sync devices failed: devices count: %d, error count: %d", len(payload.Devices), errCounter))
		} else {
			c.syncDevicesRes(nil)
		}
	}
	return nil
}

// syncDevicesRes 发送设备同步响应
func (c *connector) syncDevicesRes(err error) {
	var errMsg string
	if err != nil {
		errMsg = err.Error()
	}

	if err = c.sendWebSocketPayload(gateway.WSPayload{
		Type:  gateway.WSForSyncDevicesRes,
		Error: errMsg,
	}); err != nil {
		helper.Logger.Error("gateway plugin send sync devices res failed", zap.String("IP", c.conf.IP))
	}
}

func (c *connector) syncShadow(payload gateway.WSPayload) error {
	if len(payload.Shadow) > 0 {
		for _, device := range payload.Shadow {
			// 添加设备
			helper.DeviceShadow.AddDevice(device.ID, device.ModelName)
			// 更新点位数据
			for _, point := range device.Points {
				_ = helper.DeviceShadow.SetDevicePoint(device.ID, point.Name, point.Value)
			}
		}
	}

	c.syncShadowRes(nil)
	return nil
}

func (c *connector) syncShadowRes(err error) {
	var errMsg string
	if err != nil {
		errMsg = err.Error()
	}

	if err = c.sendWebSocketPayload(gateway.WSPayload{
		Type:  gateway.WSForSyncShadowRes,
		Error: errMsg,
	}); err != nil {
		helper.Logger.Error("gateway plugin send shadow res failed", zap.String("IP", c.conf.IP))
	}
}

// control 向子网关发送控制消息
func (c *connector) control(data plugin.DeviceData) {
	if err := c.sendWebSocketPayload(gateway.WSPayload{
		Type:       gateway.WSForControl,
		DeviceData: data,
	}); err != nil {
		helper.Logger.Error("gateway plugin send control payload failed", zap.String("IP", c.conf.IP))
	}
}

func (c *connector) controlRes(payload gateway.WSPayload) error {
	if payload.Error != "" {
		helper.Logger.Error("gateway plugin control failed", zap.String("IP", c.conf.IP), zap.String("error", payload.Error))
	} else {
		helper.Logger.Info("gateway plugin control success", zap.String("IP", c.conf.IP))
	}
	return nil
}
