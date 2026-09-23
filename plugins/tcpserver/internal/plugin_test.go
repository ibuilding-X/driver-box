package internal

import (
	"sync"
	"testing"

	"github.com/ibuilding-x/driver-box/v2/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/v2/pkg/config"
)

// newTestPlugin 创建测试用的插件
func newTestPlugin() *Plugin {
	return &Plugin{
		config: config.DeviceConfig{
			Connections: map[string]interface{}{
				"test-connection": map[string]interface{}{
					"host":        "127.0.0.1",
					"port":        0,
					"buffSize":    1024,
					"readTimeout": 30,
					"enable":      true,
					"protocolKey": "test-protocol",
				},
			},
		},
		connPool: make(map[string]*connector),
	}
}

// TestPluginInitialize 测试插件初始化
func TestPluginInitialize(t *testing.T) {
	requireListen(t)

	p := newTestPlugin()

	// 初始化插件
	p.Initialize(p.config)

	// 验证连接池已创建
	if p.connPool == nil {
		t.Fatal("connPool should not be nil after Initialize")
	}

	// 清理
	p.Destroy()
}

// TestPluginConnector 测试获取连接器
func TestPluginConnector(t *testing.T) {
	requireListen(t)

	// 注意：由于 Connector() 需要 driverbox.CoreCache() 中的设备信息，
	// 这里测试错误情况

	p := newTestPlugin()
	p.Initialize(p.config)

	// 测试获取不存在的设备
	_, err := p.Connector("nonexistent-device")
	if err == nil {
		t.Fatal("expected error for nonexistent device")
	}

	// 清理
	p.Destroy()
}

// TestPluginDestroy 测试插件销毁
func TestPluginDestroy(t *testing.T) {
	requireListen(t)

	p := newTestPlugin()

	// 初始化插件
	p.Initialize(p.config)

	// 销毁插件
	err := p.Destroy()
	if err != nil {
		t.Fatalf("Destroy failed: %v", err)
	}
}

// TestPluginMultipleConnections 测试多连接配置
func TestPluginMultipleConnections(t *testing.T) {
	requireListen(t)

	p := &Plugin{
		config: config.DeviceConfig{
			Connections: map[string]interface{}{
				"connection1": map[string]interface{}{
					"host":        "127.0.0.1",
					"port":        0,
					"buffSize":    1024,
					"readTimeout": 30,
					"enable":      true,
					"protocolKey": "protocol1",
				},
				"connection2": map[string]interface{}{
					"host":        "127.0.0.1",
					"port":        0,
					"buffSize":    2048,
					"readTimeout": 60,
					"enable":      true,
					"protocolKey": "protocol2",
				},
			},
		},
		connPool: make(map[string]*connector),
	}

	// 初始化插件
	p.Initialize(p.config)

	// 验证两个连接器都被创建
	if len(p.connPool) != 2 {
		t.Fatalf("connPool size = %d, want 2", len(p.connPool))
	}

	// 验证连接器存在
	_, ok := p.connPool["connection1"]
	if !ok {
		t.Fatal("connection1 not found in connPool")
	}
	_, ok = p.connPool["connection2"]
	if !ok {
		t.Fatal("connection2 not found in connPool")
	}

	// 清理
	p.Destroy()
}

// TestPluginDisabledConnection 测试禁用的连接
func TestPluginDisabledConnection(t *testing.T) {
	p := &Plugin{
		config: config.DeviceConfig{
			Connections: map[string]interface{}{
				"disabled-connection": map[string]interface{}{
					"host":        "127.0.0.1",
					"port":        0,
					"buffSize":    1024,
					"readTimeout": 30,
					"enable":      false, // 禁用
					"protocolKey": "test-protocol",
				},
			},
		},
		connPool: make(map[string]*connector),
	}

	// 初始化插件
	p.Initialize(p.config)

	// 禁用的连接不应该启动 listener
	if len(p.connPool) != 0 {
		t.Fatalf("connPool size = %d, want 0", len(p.connPool))
	}
}

// TestPluginConcurrentAccess 测试并发访问
func TestPluginConcurrentAccess(t *testing.T) {
	requireListen(t)

	p := newTestPlugin()
	p.Initialize(p.config)

	var wg sync.WaitGroup

	// 并发获取连接器
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			deviceId := "device" + string(rune('A'+i))
			p.Connector(deviceId)
		}(i)
	}

	wg.Wait()

	// 清理
	p.Destroy()
}

// TestPluginInitConnPool 测试连接池初始化
func TestPluginInitConnPool(t *testing.T) {
	requireListen(t)

	p := &Plugin{
		config: config.DeviceConfig{
			Connections: map[string]interface{}{
				"test-conn": map[string]interface{}{
					"host":        "127.0.0.1",
					"port":        0,
					"buffSize":    1024,
					"readTimeout": 30,
					"enable":      true,
					"protocolKey": "test-protocol",
				},
			},
		},
		connPool: make(map[string]*connector),
	}

	// 测试初始化连接池
	err := p.initConnPool()
	if err != nil {
		t.Fatalf("initConnPool failed: %v", err)
	}

	// 验证连接器已创建
	if len(p.connPool) != 1 {
		t.Fatalf("connPool size = %d, want 1", len(p.connPool))
	}

	// 验证连接器配置正确
	conn, ok := p.connPool["test-conn"]
	if !ok {
		t.Fatal("test-conn not found in connPool")
	}

	if conn.config.Host != "127.0.0.1" {
		t.Fatalf("conn host = %q, want %q", conn.config.Host, "127.0.0.1")
	}
	if conn.config.BuffSize != 1024 {
		t.Fatalf("conn buffSize = %d, want %d", conn.config.BuffSize, 1024)
	}
	if conn.config.ReadTimeout != 30 {
		t.Fatalf("conn readTimeout = %d, want %d", conn.config.ReadTimeout, 30)
	}

	// 清理
	p.Destroy()
}

// TestPluginConnectionDefaults 测试连接配置默认值
func TestPluginConnectionDefaults(t *testing.T) {
	requireListen(t)

	p := &Plugin{
		config: config.DeviceConfig{
			Connections: map[string]interface{}{
				"default-connection": map[string]interface{}{
					"host":        "127.0.0.1",
					"port":        0,
					"protocolKey": "test-protocol",
				},
			},
		},
		connPool: make(map[string]*connector),
	}

	if err := p.initConnPool(); err != nil {
		t.Fatalf("initConnPool failed: %v", err)
	}
	defer p.Destroy()

	conn, ok := p.connPool["default-connection"]
	if !ok {
		t.Fatal("default-connection not found")
	}
	if conn.config.BuffSize != 1024 {
		t.Fatalf("default buffSize = %d, want 1024", conn.config.BuffSize)
	}
	if conn.config.ReadTimeout != 30 {
		t.Fatalf("default readTimeout = %d, want 30", conn.config.ReadTimeout)
	}
	if !conn.config.Enable {
		t.Fatal("omitted enable should default to true")
	}
}

// TestPluginInitConnPoolInvalidConfig 测试无效配置
func TestPluginInitConnPoolInvalidConfig(t *testing.T) {
	p := &Plugin{
		config: config.DeviceConfig{
			Connections: map[string]interface{}{
				"invalid-conn": "invalid config", // 无效的配置
			},
		},
		connPool: make(map[string]*connector),
	}

	// 测试初始化连接池
	err := p.initConnPool()
	if err == nil {
		t.Fatal("expected error for invalid config")
	}
}

// TestPluginProtocolName 测试协议名称
func TestPluginProtocolName(t *testing.T) {
	if ProtocolName != "tcp_server" {
		t.Fatalf("ProtocolName = %q, want %q", ProtocolName, "tcp_server")
	}
}

// TestPluginBaseConnection 测试基础连接配置
func TestPluginBaseConnection(t *testing.T) {
	config := plugin.BaseConnection{
		ConnectionKey: "test-key",
		ProtocolKey:   "test-protocol",
		Enable:        true,
		Discover:      true,
		Virtual:       false,
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
	if config.Virtual {
		t.Fatal("Virtual should be false")
	}
}
