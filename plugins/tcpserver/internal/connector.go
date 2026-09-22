package internal

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/ibuilding-x/driver-box/v2/driverbox"
	"github.com/ibuilding-x/driver-box/v2/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/v2/pkg/library"
	"go.uber.org/zap"
)

type connector struct {
	config   connectorConfig
	listener net.Listener

	// 服务生命周期状态。listener 关闭后，accept 与连接处理协程需要能够及时退出。
	lifecycleMu sync.Mutex
	closed      bool
	stop        chan struct{}
	acceptWG    sync.WaitGroup
	connWG      sync.WaitGroup

	// 设备与连接的映射 (deviceId -> net.Conn)
	deviceMappingConn *sync.Map
	// 连接与设备的映射 (net.Conn -> []string)
	connMappingDevice *sync.Map
	// 当前活跃连接，Release 时需要全部关闭
	activeConns *sync.Map
}

// connectorConfig 连接器配置
type connectorConfig struct {
	plugin.BaseConnection
	Host        string `json:"host"`
	Port        uint16 `json:"port"`
	BuffSize    uint   `json:"buffSize"`
	ReadTimeout int    `json:"readTimeout"` // 读取超时，单位秒，默认30秒
}

// encodeStruct 编码后的数据结构
type encodeStruct struct {
	conn    net.Conn
	payload string
}

// protoData 协议数据
type protoData struct {
	RemoteAddr string `json:"remoteAddr,omitempty"`
	Event      string `json:"event,omitempty"` // 当前实现发送 read
	Raw        string `json:"raw"`
}

// newConnector 创建连接器，并初始化生命周期资源。
func newConnector(config connectorConfig) *connector {
	return &connector{
		config:            config,
		stop:              make(chan struct{}),
		deviceMappingConn: &sync.Map{},
		connMappingDevice: &sync.Map{},
		activeConns:       &sync.Map{},
	}
}

// ToJSON 协议数据转 json 字符串
func (pd protoData) ToJSON() string {
	b, _ := json.Marshal(pd)
	return string(b)
}

// Send 发送编码后的数据到设备
func (c *connector) Send(raw interface{}) (err error) {
	data, ok := raw.(encodeStruct)
	if !ok {
		return errors.New("invalid encode data type")
	}
	_, err = data.conn.Write([]byte(data.payload))
	return err
}

// Release 释放资源，关闭 listener、活跃连接，并等待相关协程退出。
func (c *connector) Release() (err error) {
	c.lifecycleMu.Lock()
	if !c.closed {
		c.closed = true
		close(c.stop)
		if c.listener != nil {
			err = c.listener.Close()
		}
	}
	listener := c.listener
	c.lifecycleMu.Unlock()

	// 未启动服务的连接器没有需要等待的协程。
	if listener == nil {
		return err
	}

	c.acceptWG.Wait()
	c.activeConns.Range(func(_, value any) bool {
		_ = value.(net.Conn).Close()
		return true
	})
	c.connWG.Wait()
	return err
}

// startServer 启动 TCP 服务
func (c *connector) startServer() (err error) {
	addr := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)

	c.lifecycleMu.Lock()
	if c.closed {
		c.lifecycleMu.Unlock()
		return errors.New("connector is released")
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		c.lifecycleMu.Unlock()
		return err
	}
	c.listener = listener
	// 必须在持有 lifecycleMu 时登记 accept 协程，避免 Release 在 Add 之前完成等待。
	c.acceptWG.Add(1)
	c.lifecycleMu.Unlock()

	go c.acceptLoop(listener, addr)
	return nil
}

func (c *connector) acceptLoop(listener net.Listener, addr string) {
	defer c.acceptWG.Done()

	driverbox.Log().Info("Listening and serving TCP", zap.String("addr", addr))
	for {
		conn, err := listener.Accept()
		if err != nil {
			// 主动 Release 时不需要按异常记录。
			select {
			case <-c.stop:
				driverbox.Log().Debug("TCP listener is closed", zap.String("addr", addr))
			default:
				driverbox.Log().Error("TCP accept connection error", zap.Error(err))
			}
			break
		}

		driverbox.Log().Debug("tcp client is connected", zap.String("remoteAddr", conn.RemoteAddr().String()))

		// 先登记连接，再启动处理协程，保证 Release 等待 accept 协程后能看到所有活跃连接。
		c.activeConns.Store(conn, conn)
		c.connWG.Add(1)
		go c.serveConn(conn)
	}
	driverbox.Log().Warn("End listening and serving TCP", zap.String("addr", addr))
}

func (c *connector) serveConn(conn net.Conn) {
	defer c.connWG.Done()
	defer c.activeConns.Delete(conn)
	c.handleConn(conn)
}

func (c *connector) readTimeout() time.Duration {
	if c.config.ReadTimeout <= 0 {
		return 30 * time.Second
	}
	return time.Duration(c.config.ReadTimeout) * time.Second
}

func (c *connector) bufferSize() int {
	if c.config.BuffSize == 0 {
		return 1024
	}
	return int(c.config.BuffSize)
}

// handleConn 处理 TCP 连接
func (c *connector) handleConn(conn net.Conn) {
	defer func() {
		// 清理连接映射
		c.cleanupConn(conn)
		conn.Close()
	}()

	buf := make([]byte, c.bufferSize())
	reader := bufio.NewReader(conn)

	for {
		_ = conn.SetReadDeadline(time.Now().Add(c.readTimeout()))

		n, err := reader.Read(buf[:])
		if err != nil {
			driverbox.Log().Debug("tcp connection read error", zap.Error(err), zap.String("remoteAddr", conn.RemoteAddr().String()))
			break
		}

		raw := string(buf[:n])
		data := protoData{
			RemoteAddr: conn.RemoteAddr().String(),
			Event:      "read",
			Raw:        raw,
		}

		// 接收数据，调用 Lua 解码
		if res, err := c.Decode(data.ToJSON()); err != nil {
			driverbox.Log().Error("tcp_server decode error", zap.Error(err), zap.String("remoteAddr", conn.RemoteAddr().String()))
		} else {
			// 更新设备与连接的映射关系
			for i := range res {
				// 解析设备ID，支持通过 deviceKey 匹配
				deviceId := c.resolveDeviceId(res[i])
				if deviceId == "" {
					driverbox.Log().Warn("cannot resolve device id",
						zap.String("rawDeviceId", res[i].ID),
						zap.String("remoteAddr", conn.RemoteAddr().String()))
					continue
				}
				// range 会复制切片元素；必须写回切片，后续发现事件和导出才能使用解析后的设备 ID。
				res[i].ID = deviceId
				c.updateMapping(deviceId, conn)
			}
			// 自动发现设备
			plugin.WrapperDiscoverEvent(res, c.config.ConnectionKey, ProtocolName)
			// 导出数据
			driverbox.Export(res)
		}
	}
}

// resolveDeviceId 解析设备ID
// 支持两种方式：
// 1. 直接使用 Lua 返回的 ID（如果配置中存在）
// 2. 通过 device_key 匹配 Properties 中的 device_key
func (c *connector) resolveDeviceId(deviceData plugin.DeviceData) string {
	// 方式1：直接使用 ID
	if deviceData.ID != "" {
		dev, exists := driverbox.CoreCache().GetDevice(deviceData.ID)
		if exists {
			// 设备 ID 已存在时必须属于当前连接，避免不同连接之间的设备互相抢占映射。
			if dev.ConnectionKey == c.config.ConnectionKey {
				return deviceData.ID
			}
			// 已知但属于其他连接的设备不能继续回退为原 ID。
			return ""
		}
	}

	// 方式2：通过 device_key 匹配 Properties 中的 device_key
	deviceKey := c.extractDeviceKey(deviceData)
	if deviceKey == "" {
		return deviceData.ID
	}

	// 遍历配置中的设备，查找 Properties 中 device_key 匹配的设备
	devices := driverbox.CoreCache().Devices()
	for _, dev := range devices {
		// 检查是否是当前连接的设备
		if dev.ConnectionKey != c.config.ConnectionKey {
			continue
		}
		// 检查 Properties 中的 device_key
		if dev.Properties != nil && dev.Properties["device_key"] == deviceKey {
			return dev.ID
		}
	}

	return ""
}

// extractDeviceKey 从设备数据中提取 device_key
func (c *connector) extractDeviceKey(deviceData plugin.DeviceData) string {
	for _, v := range deviceData.Values {
		if v.PointName == "device_key" {
			if key, ok := v.Value.(string); ok {
				return key
			}
		}
	}
	return ""
}

// updateMapping 更新设备与连接的映射关系
func (c *connector) updateMapping(deviceId string, conn net.Conn) {
	// 更新设备到连接的映射
	preConn, _ := c.deviceMappingConn.Swap(deviceId, conn)
	if preConn == conn {
		return
	}

	// 在新连接中加入当前设备
	devices, ok := c.connMappingDevice.Load(conn)
	if ok {
		deviceList := devices.([]string)
		// 检查是否已存在
		for _, d := range deviceList {
			if d == deviceId {
				return
			}
		}
		deviceList = append(deviceList, deviceId)
		c.connMappingDevice.Store(conn, deviceList)
	} else {
		c.connMappingDevice.Store(conn, []string{deviceId})
	}
}

// cleanupConn 清理连接断开时的映射
func (c *connector) cleanupConn(conn net.Conn) {
	// 移除连接到设备的映射
	devices, ok := c.connMappingDevice.LoadAndDelete(conn)
	if !ok {
		return
	}

	// 将设备设置为离线
	for _, device := range devices.([]string) {
		// 若移除失败，说明当前设备最近一次是通过其他 TCP 连接上报的，则无需处理
		// 否则，将该设备设置为：离线
		deleted := c.deviceMappingConn.CompareAndDelete(device, conn)
		if deleted {
			// 尝试设置设备离线，如果 Shadow 未初始化则跳过
			shadow := driverbox.Shadow()
			if shadow != nil {
				_ = shadow.SetOffline(device)
			}
			// 尝试记录日志，如果 Logger 未初始化则跳过
			logger := driverbox.Log()
			if logger != nil {
				logger.Info("device offline due to connection closed",
					zap.String("device", device),
					zap.String("remoteAddr", conn.RemoteAddr().String()))
			}
		}
	}
}

// Encode 编码数据
func (c *connector) Encode(deviceId string, mode plugin.EncodeMode, values ...plugin.PointData) (res interface{}, err error) {
	payload, err := library.Protocol().Encode(c.config.ProtocolKey, library.ProtocolEncodeRequest{
		DeviceId: deviceId,
		Mode:     mode,
		Points:   values,
	})
	if err != nil {
		return nil, err
	}

	// 获取设备对应的连接
	conn, ok := c.deviceMappingConn.Load(deviceId)
	if !ok {
		return nil, fmt.Errorf("device %s is disconnected", deviceId)
	}

	return encodeStruct{
		payload: payload,
		conn:    conn.(net.Conn),
	}, nil
}

// SendToDevice 主动发送原始数据到指定设备
// 该方法不需要通过 Lua encode，直接发送原始数据
// 适用于：
//   - 心跳响应
//   - 自定义协议命令
//   - 测试/调试场景
func (c *connector) SendToDevice(deviceId string, data []byte) error {
	// 获取设备对应的连接
	conn, ok := c.deviceMappingConn.Load(deviceId)
	if !ok {
		return fmt.Errorf("device %s is disconnected", deviceId)
	}

	_, err := conn.(net.Conn).Write(data)
	return err
}

// SendToAll 主动发送原始数据到所有已连接的设备
// 适用于广播场景
func (c *connector) SendToAll(data []byte) (sent int, err error) {
	c.deviceMappingConn.Range(func(key, value interface{}) bool {
		conn := value.(net.Conn)
		if _, writeErr := conn.Write(data); writeErr != nil {
			driverbox.Log().Error("failed to send to device",
				zap.String("deviceId", key.(string)),
				zap.Error(writeErr))
		} else {
			sent++
		}
		return true
	})
	return sent, nil
}

// GetConnectedDevices 获取所有已连接的设备ID列表
func (c *connector) GetConnectedDevices() []string {
	var devices []string
	c.deviceMappingConn.Range(func(key, value interface{}) bool {
		devices = append(devices, key.(string))
		return true
	})
	return devices
}

// IsDeviceConnected 检查设备是否已连接
func (c *connector) IsDeviceConnected(deviceId string) bool {
	_, ok := c.deviceMappingConn.Load(deviceId)
	return ok
}

// Decode 解码数据
func (c *connector) Decode(raw interface{}) (res []plugin.DeviceData, err error) {
	return library.Protocol().Decode(c.config.ProtocolKey, raw)
}
