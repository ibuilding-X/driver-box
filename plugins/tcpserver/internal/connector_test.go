package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/ibuilding-x/driver-box/v2/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/v2/internal/logger"
)

func init() {
	// 初始化 logger 以避免测试中的 panic
	if logger.Logger == nil {
		logger.InitLogger("", "debug")
	}
}

// mockConn 模拟网络连接
type mockConn struct {
	mu         sync.Mutex
	written    []byte
	closed     bool
	remoteAddr string
}

func newMockConn(addr string) *mockConn {
	return &mockConn{remoteAddr: addr}
}

func (m *mockConn) Read(b []byte) (n int, err error) { return 0, nil }
func (m *mockConn) Write(b []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.written = append(m.written, b...)
	return len(b), nil
}
func (m *mockConn) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}
func (m *mockConn) LocalAddr() net.Addr { return &net.TCPAddr{} }
func (m *mockConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP(m.remoteAddr), Port: 12345}
}
func (m *mockConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }

func (m *mockConn) GetWritten() []byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]byte, len(m.written))
	copy(result, m.written)
	return result
}

func (m *mockConn) IsClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}

// newTestConnector 创建测试用的连接器
func newTestConnector() *connector {
	return newConnector(connectorConfig{
		BaseConnection: plugin.BaseConnection{
			ConnectionKey: "test-connection",
			ProtocolKey:   "test-protocol",
			Enable:        true,
		},
		Host:        "127.0.0.1",
		Port:        0, // 随机端口
		BuffSize:    1024,
		ReadTimeout: 30,
	})
}

// requireListen 在当前环境禁止创建 TCP socket 时跳过依赖真实 listener 的测试。
func requireListen(t *testing.T) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("TCP listener is unavailable: %v", err)
	}
	_ = listener.Close()
}

// TestProtoDataToJSON 测试协议数据转换为JSON
func TestProtoDataToJSON(t *testing.T) {
	tests := []struct {
		name     string
		data     protoData
		expected map[string]interface{}
	}{
		{
			name: "basic data",
			data: protoData{
				RemoteAddr: "127.0.0.1:12345",
				Event:      "read",
				Raw:        "test raw data",
			},
			expected: map[string]interface{}{
				"remoteAddr": "127.0.0.1:12345",
				"event":      "read",
				"raw":        "test raw data",
			},
		},
		{
			name: "minimal data",
			data: protoData{
				Raw: "minimal",
			},
			expected: map[string]interface{}{
				"raw": "minimal",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonStr := tt.data.ToJSON()
			var result map[string]interface{}
			if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
				t.Fatalf("failed to unmarshal JSON: %v", err)
			}

			for key, expectedValue := range tt.expected {
				actualValue, ok := result[key]
				if !ok {
					t.Errorf("missing key %s in result", key)
					continue
				}
				if actualValue != expectedValue {
					t.Errorf("key %s: expected %v, got %v", key, expectedValue, actualValue)
				}
			}
		})
	}
}

// TestUpdateMapping 测试设备映射更新
func TestUpdateMapping(t *testing.T) {
	c := newTestConnector()
	conn1 := newMockConn("192.168.1.1")
	conn2 := newMockConn("192.168.1.2")

	// 测试添加新设备
	c.updateMapping("device1", conn1)

	// 验证设备到连接的映射
	mappedConn, ok := c.deviceMappingConn.Load("device1")
	if !ok {
		t.Fatal("device1 not found in deviceMappingConn")
	}
	if mappedConn != conn1 {
		t.Fatal("device1 mapped to wrong connection")
	}

	// 验证连接到设备的映射
	devices, ok := c.connMappingDevice.Load(conn1)
	if !ok {
		t.Fatal("conn1 not found in connMappingDevice")
	}
	deviceList := devices.([]string)
	if len(deviceList) != 1 || deviceList[0] != "device1" {
		t.Fatalf("conn1 devices = %v, want [device1]", deviceList)
	}

	// 测试同一设备更新连接
	c.updateMapping("device1", conn2)

	mappedConn, ok = c.deviceMappingConn.Load("device1")
	if !ok {
		t.Fatal("device1 not found after update")
	}
	if mappedConn != conn2 {
		t.Fatal("device1 should be mapped to conn2 after update")
	}

	// 测试同一连接添加多个设备
	c.updateMapping("device2", conn2)

	devices, ok = c.connMappingDevice.Load(conn2)
	if !ok {
		t.Fatal("conn2 not found in connMappingDevice")
	}
	deviceList = devices.([]string)
	if len(deviceList) != 2 {
		t.Fatalf("conn2 devices count = %d, want 2", len(deviceList))
	}

	// 测试重复添加同一设备
	c.updateMapping("device2", conn2)
	devices, ok = c.connMappingDevice.Load(conn2)
	if !ok {
		t.Fatal("conn2 not found after duplicate add")
	}
	deviceList = devices.([]string)
	if len(deviceList) != 2 {
		t.Fatalf("conn2 devices count after duplicate = %d, want 2", len(deviceList))
	}
}

// TestCleanupConn 测试连接清理
func TestCleanupConn(t *testing.T) {
	c := newTestConnector()
	conn1 := newMockConn("192.168.1.1")
	conn2 := newMockConn("192.168.1.2")

	// 添加设备映射
	c.updateMapping("device1", conn1)
	c.updateMapping("device2", conn1)
	c.updateMapping("device3", conn2)

	// 清理 conn1
	c.cleanupConn(conn1)

	// 验证 conn1 的设备映射已删除
	_, ok := c.connMappingDevice.Load(conn1)
	if ok {
		t.Fatal("conn1 should be removed from connMappingDevice")
	}

	// 验证 device1 和 device2 的连接映射已删除
	_, ok = c.deviceMappingConn.Load("device1")
	if ok {
		t.Fatal("device1 should be removed from deviceMappingConn")
	}
	_, ok = c.deviceMappingConn.Load("device2")
	if ok {
		t.Fatal("device2 should be removed from deviceMappingConn")
	}

	// 验证 device3 仍然存在
	_, ok = c.deviceMappingConn.Load("device3")
	if !ok {
		t.Fatal("device3 should still exist in deviceMappingConn")
	}
}

// TestEncode 测试编码功能
func TestEncode(t *testing.T) {
	c := newTestConnector()
	conn := newMockConn("192.168.1.1")

	// 添加设备映射
	c.updateMapping("device1", conn)

	// 测试编码 - 设备未连接
	t.Run("device not connected", func(t *testing.T) {
		_, err := c.Encode("nonexistent", plugin.WriteMode, plugin.PointData{
			PointName: "test",
			Value:     1,
		})
		if err == nil {
			t.Fatal("expected error for nonexistent device")
		}
		// 注意：由于需要实际的 Lua 脚本，这里可能会返回 "lua script not found" 错误
		// 我们只验证确实返回了错误
		t.Logf("Encode error (expected): %v", err)
	})
}

// TestSend 测试发送功能
func TestSend(t *testing.T) {
	c := newTestConnector()
	conn := newMockConn("192.168.1.1")

	// 添加设备映射
	c.updateMapping("device1", conn)

	// 测试发送有效数据
	testPayload := "test payload data"
	encodeData := encodeStruct{
		conn:    conn,
		payload: testPayload,
	}

	err := c.Send(encodeData)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// 验证数据已发送
	written := conn.GetWritten()
	if string(written) != testPayload {
		t.Fatalf("written data = %q, want %q", string(written), testPayload)
	}

	// 测试发送无效类型
	err = c.Send("invalid type")
	if err == nil {
		t.Fatal("expected error for invalid type")
	}
	if err.Error() != "invalid encode data type" {
		t.Fatalf("unexpected error message: %v", err)
	}
}

// TestConcurrentMapping 测试并发映射操作
func TestConcurrentMapping(t *testing.T) {
	c := newTestConnector()
	var wg sync.WaitGroup

	// 并发添加设备
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			conn := newMockConn(fmt.Sprintf("192.168.1.%d", i))
			deviceId := fmt.Sprintf("device%d", i)
			c.updateMapping(deviceId, conn)
		}(i)
	}
	wg.Wait()

	// 验证所有设备都已添加
	for i := 0; i < 100; i++ {
		deviceId := fmt.Sprintf("device%d", i)
		_, ok := c.deviceMappingConn.Load(deviceId)
		if !ok {
			t.Fatalf("device %s not found after concurrent add", deviceId)
		}
	}

	// 并发清理连接
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			conn := newMockConn(fmt.Sprintf("192.168.1.%d", i))
			// 先添加设备到连接
			deviceId := fmt.Sprintf("device%d", i)
			c.updateMapping(deviceId, conn)
			// 然后清理连接
			c.cleanupConn(conn)
		}(i)
	}
	wg.Wait()

	// 验证前50个设备已被清理
	for i := 0; i < 50; i++ {
		deviceId := fmt.Sprintf("device%d", i)
		_, ok := c.deviceMappingConn.Load(deviceId)
		if ok {
			t.Fatalf("device %s should be cleaned up", deviceId)
		}
	}
}

// TestRelease 测试资源释放
func TestRelease(t *testing.T) {
	c := newTestConnector()

	// 测试没有 listener 的情况
	err := c.Release()
	if err != nil {
		t.Fatalf("Release failed: %v", err)
	}

	requireListen(t)

	// 创建一个真实的 TCP listener 用于测试
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}
	c = newTestConnector()
	c.listener = listener

	// 释放资源
	err = c.Release()
	if err != nil {
		t.Fatalf("Release failed: %v", err)
	}

	// 验证 listener 已关闭
	_, err = listener.Accept()
	if err == nil {
		t.Fatal("listener should be closed")
	}
}

// TestStartServer 测试服务器启动
func TestStartServer(t *testing.T) {
	requireListen(t)

	c := newTestConnector()

	// 使用随机端口
	c.config.Port = 0

	err := c.startServer()
	if err != nil {
		t.Fatalf("startServer failed: %v", err)
	}

	// 确保服务器启动
	if c.listener == nil {
		t.Fatal("listener should not be nil after startServer")
	}

	// 获取实际监听地址
	addr := c.listener.Addr().(*net.TCPAddr)
	t.Logf("Server listening on port %d", addr.Port)

	// 清理
	if err := c.Release(); err != nil {
		t.Fatalf("Release failed: %v", err)
	}
	// Release 应该可重复调用
	if err := c.Release(); err != nil {
		t.Fatalf("second Release failed: %v", err)
	}
}

// TestHandleConn 测试连接处理
func TestHandleConn(t *testing.T) {
	// 注意：handleConn 会调用 Decode，需要实际的 Lua 脚本支持
	// 这里测试基本的连接处理逻辑

	c := newTestConnector()

	// 创建一个 pipe 模拟网络连接
	server, client := net.Pipe()
	defer client.Close()

	// 启动一个 goroutine 处理连接
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		// handleConn 会因为读取超时而退出
		c.handleConn(server)
	}()

	// 等待一段时间让连接处理
	time.Sleep(100 * time.Millisecond)

	// 关闭客户端连接
	client.Close()

	// 等待处理完成
	wg.Wait()
}

// TestEncodeStruct 测试编码结构体
func TestEncodeStruct(t *testing.T) {
	conn := newMockConn("192.168.1.1")
	payload := "test payload"

	data := encodeStruct{
		conn:    conn,
		payload: payload,
	}

	if data.conn != conn {
		t.Fatal("conn should match")
	}
	if data.payload != payload {
		t.Fatal("payload should match")
	}
}

// TestSendToDevice 测试主动发送到指定设备
func TestSendToDevice(t *testing.T) {
	c := newTestConnector()
	conn := newMockConn("192.168.1.1")

	// 添加设备映射
	c.updateMapping("device1", conn)

	// 测试发送成功
	testData := []byte("test data for device1")
	err := c.SendToDevice("device1", testData)
	if err != nil {
		t.Fatalf("SendToDevice failed: %v", err)
	}

	// 验证数据已发送
	written := conn.GetWritten()
	if string(written) != string(testData) {
		t.Fatalf("written data = %q, want %q", string(written), string(testData))
	}

	// 测试设备未连接
	err = c.SendToDevice("nonexistent", testData)
	if err == nil {
		t.Fatal("expected error for nonexistent device")
	}
	if err.Error() != "device nonexistent is disconnected" {
		t.Fatalf("unexpected error message: %v", err)
	}
}

// TestSendToAll 测试广播到所有设备
func TestSendToAll(t *testing.T) {
	c := newTestConnector()
	conn1 := newMockConn("192.168.1.1")
	conn2 := newMockConn("192.168.1.2")
	conn3 := newMockConn("192.168.1.3")

	// 添加设备映射
	c.updateMapping("device1", conn1)
	c.updateMapping("device2", conn2)
	c.updateMapping("device3", conn3)

	// 测试广播
	testData := []byte("broadcast data")
	sent, err := c.SendToAll(testData)
	if err != nil {
		t.Fatalf("SendToAll failed: %v", err)
	}
	if sent != 3 {
		t.Fatalf("sent = %d, want 3", sent)
	}

	// 验证所有设备都收到了数据
	for _, conn := range []*mockConn{conn1, conn2, conn3} {
		written := conn.GetWritten()
		if string(written) != string(testData) {
			t.Fatalf("device %s: written data = %q, want %q", conn.remoteAddr, string(written), string(testData))
		}
	}

	// 测试空连接池
	c2 := newTestConnector()
	sent, err = c2.SendToAll(testData)
	if err != nil {
		t.Fatalf("SendToAll failed for empty pool: %v", err)
	}
	if sent != 0 {
		t.Fatalf("sent = %d for empty pool, want 0", sent)
	}
}

// TestGetConnectedDevices 测试获取已连接设备列表
func TestGetConnectedDevices(t *testing.T) {
	c := newTestConnector()

	// 初始状态应该为空
	devices := c.GetConnectedDevices()
	if len(devices) != 0 {
		t.Fatalf("initial devices count = %d, want 0", len(devices))
	}

	// 添加设备
	c.updateMapping("device1", newMockConn("192.168.1.1"))
	c.updateMapping("device2", newMockConn("192.168.1.2"))
	c.updateMapping("device3", newMockConn("192.168.1.3"))

	// 获取设备列表
	devices = c.GetConnectedDevices()
	if len(devices) != 3 {
		t.Fatalf("devices count = %d, want 3", len(devices))
	}

	// 验证设备存在
	deviceMap := make(map[string]bool)
	for _, d := range devices {
		deviceMap[d] = true
	}
	for _, expected := range []string{"device1", "device2", "device3"} {
		if !deviceMap[expected] {
			t.Fatalf("missing device %s", expected)
		}
	}
}

// TestIsDeviceConnected 测试设备连接状态检查
func TestIsDeviceConnected(t *testing.T) {
	c := newTestConnector()

	// 初始状态应该未连接
	if c.IsDeviceConnected("device1") {
		t.Fatal("device1 should not be connected initially")
	}

	// 添加设备
	c.updateMapping("device1", newMockConn("192.168.1.1"))

	// 验证已连接
	if !c.IsDeviceConnected("device1") {
		t.Fatal("device1 should be connected after updateMapping")
	}

	// 验证其他设备未连接
	if c.IsDeviceConnected("device2") {
		t.Fatal("device2 should not be connected")
	}
}

// TestExtractDeviceKey 测试提取 device_key
func TestExtractDeviceKey(t *testing.T) {
	c := newTestConnector()

	// 测试从 values 中提取 device_key
	deviceData := plugin.DeviceData{
		ID: "test_device",
		Values: []plugin.PointData{
			{PointName: "device_key", Value: "PILE001"},
			{PointName: "status", Value: "charging"},
		},
	}

	deviceKey := c.extractDeviceKey(deviceData)
	if deviceKey != "PILE001" {
		t.Fatalf("extractDeviceKey = %q, want %q", deviceKey, "PILE001")
	}

	// 测试没有 device_key 的情况
	deviceData2 := plugin.DeviceData{
		ID: "test_device",
		Values: []plugin.PointData{
			{PointName: "status", Value: "charging"},
		},
	}

	deviceKey2 := c.extractDeviceKey(deviceData2)
	if deviceKey2 != "" {
		t.Fatalf("extractDeviceKey = %q, want empty string", deviceKey2)
	}

	// 测试 device_key 不是字符串的情况
	deviceData3 := plugin.DeviceData{
		ID: "test_device",
		Values: []plugin.PointData{
			{PointName: "device_key", Value: 12345},
		},
	}

	deviceKey3 := c.extractDeviceKey(deviceData3)
	if deviceKey3 != "" {
		t.Fatalf("extractDeviceKey = %q, want empty string for non-string value", deviceKey3)
	}
}

// TestConnectorConfig 测试连接器配置
func TestConnectorConfig(t *testing.T) {
	config := connectorConfig{
		BaseConnection: plugin.BaseConnection{
			ConnectionKey: "test-key",
			ProtocolKey:   "test-protocol",
			Enable:        true,
			Discover:      true,
		},
		Host:        "0.0.0.0",
		Port:        5000,
		BuffSize:    2048,
		ReadTimeout: 60,
	}

	if config.ConnectionKey != "test-key" {
		t.Fatalf("ConnectionKey = %q, want %q", config.ConnectionKey, "test-key")
	}
	if config.ProtocolKey != "test-protocol" {
		t.Fatalf("ProtocolKey = %q, want %q", config.ProtocolKey, "test-protocol")
	}
	if !config.Enable {
		t.Fatal("Enable should be true")
	}
	if !config.Discover {
		t.Fatal("Discover should be true")
	}
	if config.Host != "0.0.0.0" {
		t.Fatalf("Host = %q, want %q", config.Host, "0.0.0.0")
	}
	if config.Port != 5000 {
		t.Fatalf("Port = %d, want %d", config.Port, 5000)
	}
	if config.BuffSize != 2048 {
		t.Fatalf("BuffSize = %d, want %d", config.BuffSize, 2048)
	}
	if config.ReadTimeout != 60 {
		t.Fatalf("ReadTimeout = %d, want %d", config.ReadTimeout, 60)
	}
}

// TestParseDecodeResult_LegacyArray 测试传统 JSON 数组格式
func TestParseDecodeResult_LegacyArray(t *testing.T) {
	jsonStr := `[
		{
			"id": "device001",
			"values": [
				{"name": "temperature", "value": 25.5},
				{"name": "humidity", "value": 60}
			]
		}
	]`

	devices, reply, err := parseDecodeResult(jsonStr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(devices) != 1 {
		t.Fatalf("expected 1 device, got %d", len(devices))
	}
	if devices[0].ID != "device001" {
		t.Fatalf("expected ID device001, got %s", devices[0].ID)
	}
	if len(devices[0].Values) != 2 {
		t.Fatalf("expected 2 values, got %d", len(devices[0].Values))
	}
	if reply != nil {
		t.Fatalf("expected nil reply, got %v", reply)
	}
}

// TestParseDecodeResult_ExtendedObject_TextReply 测试扩展对象带文本响应
func TestParseDecodeResult_ExtendedObject_TextReply(t *testing.T) {
	jsonStr := `{
		"devices": [
			{
				"id": "device001",
				"values": [{"name": "status", "value": "online"}]
			}
		],
		"reply": "ACK\n"
	}`

	devices, reply, err := parseDecodeResult(jsonStr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(devices) != 1 || devices[0].ID != "device001" {
		t.Fatalf("devices = %v, want 1 device with ID device001", devices)
	}
	expectedReply := []byte("ACK\n")
	if !bytes.Equal(reply, expectedReply) {
		t.Fatalf("reply = %q, want %q", reply, expectedReply)
	}
}

// TestParseDecodeResult_ExtendedObject_HexReply 测试扩展对象带十六进制响应
func TestParseDecodeResult_ExtendedObject_HexReply(t *testing.T) {
	tests := []struct {
		name     string
		hexStr   string
		expected []byte
	}{
		{
			name:     "plain hex",
			hexStr:   "AAF5000100",
			expected: []byte{0xAA, 0xF5, 0x00, 0x01, 0x00},
		},
		{
			name:     "hex with spaces and prefix",
			hexStr:   "0xAA 0xF5 0x00 0x01 0x00",
			expected: []byte{0xAA, 0xF5, 0x00, 0x01, 0x00},
		},
		{
			name:     "lowercase hex with whitespace",
			hexStr:   " aa  f5 00 01 00 \n",
			expected: []byte{0xAA, 0xF5, 0x00, 0x01, 0x00},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonStr := fmt.Sprintf(`{"replyHex": %q}`, tt.hexStr)
			devices, reply, err := parseDecodeResult(jsonStr)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(devices) != 0 {
				t.Fatalf("expected 0 devices, got %d", len(devices))
			}
			if !bytes.Equal(reply, tt.expected) {
				t.Fatalf("reply = %X, want %X", reply, tt.expected)
			}
		})
	}
}

// TestParseDecodeResult_ExtendedObject_Base64Reply 测试扩展对象带 Base64 响应
func TestParseDecodeResult_ExtendedObject_Base64Reply(t *testing.T) {
	jsonStr := `{
		"devices": [
			{"id": "device001"}
		],
		"replyBase64": "qvUA"
	}`

	devices, reply, err := parseDecodeResult(jsonStr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(devices) != 1 {
		t.Fatalf("expected 1 device, got %d", len(devices))
	}
	expectedReply := []byte{0xAA, 0xF5, 0x00}
	if !bytes.Equal(reply, expectedReply) {
		t.Fatalf("reply = %X, want %X", reply, expectedReply)
	}
}

// TestParseDecodeResult_HeartbeatOnly 测试仅响应无点位数据
func TestParseDecodeResult_HeartbeatOnly(t *testing.T) {
	jsonStr := `{"reply": "PONG"}`

	devices, reply, err := parseDecodeResult(jsonStr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(devices) != 0 {
		t.Fatalf("expected 0 devices, got %d", len(devices))
	}
	if string(reply) != "PONG" {
		t.Fatalf("reply = %q, want PONG", string(reply))
	}
}

// TestParseDecodeResult_EmptyAndNull 测试空字符串及 null
func TestParseDecodeResult_EmptyAndNull(t *testing.T) {
	for _, input := range []string{"", "   ", "null", "[]"} {
		devices, reply, err := parseDecodeResult(input)
		if err != nil {
			t.Fatalf("input %q: unexpected error: %v", input, err)
		}
		if len(devices) != 0 {
			t.Fatalf("input %q: expected 0 devices, got %d", input, len(devices))
		}
		if reply != nil {
			t.Fatalf("input %q: expected nil reply, got %v", input, reply)
		}
	}
}

// TestParseDecodeResult_SingleDevice 兼容单设备对象格式
func TestParseDecodeResult_SingleDevice(t *testing.T) {
	jsonStr := `{
		"id": "dev_single",
		"values": [{"name": "power", "value": 100}]
	}`

	devices, reply, err := parseDecodeResult(jsonStr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(devices) != 1 || devices[0].ID != "dev_single" {
		t.Fatalf("devices = %v, want 1 device with ID dev_single", devices)
	}
	if reply != nil {
		t.Fatalf("expected nil reply, got %v", reply)
	}
}

// TestParseDecodeResult_InvalidFormat 测试异常格式
func TestParseDecodeResult_InvalidFormat(t *testing.T) {
	// 非法 JSON 数组
	_, _, err := parseDecodeResult("[invalid json")
	if err == nil {
		t.Fatal("expected error for invalid JSON array")
	}

	// 非法 JSON 对象
	_, _, err = parseDecodeResult("{invalid json")
	if err == nil {
		t.Fatal("expected error for invalid JSON object")
	}

	// 非法十六进制字符串
	_, _, err = parseDecodeResult(`{"replyHex": "NOT_A_HEX"}`)
	if err == nil {
		t.Fatal("expected error for invalid replyHex")
	}

	// 非法 Base64 字符串
	_, _, err = parseDecodeResult(`{"replyBase64": "!!!not_base64!!!"}`)
	if err == nil {
		t.Fatal("expected error for invalid replyBase64")
	}

	// 不支持的格式（纯普通字符串）
	_, _, err = parseDecodeResult("plain text")
	if err == nil {
		t.Fatal("expected error for plain text")
	}
}

