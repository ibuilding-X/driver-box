package internal

import (
	"errors"
	"sync"

	"github.com/ibuilding-x/driver-box/v2/driverbox"
	"github.com/ibuilding-x/driver-box/v2/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/v2/pkg/config"
	"github.com/ibuilding-x/driver-box/v2/pkg/convutil"
	"go.uber.org/zap"
)

const ProtocolName = "tcp_server"

type Plugin struct {
	mu       sync.RWMutex
	config   config.DeviceConfig
	connPool map[string]*connector // 连接池，key为ConnectionKey
}

// Initialize 插件初始化
func (p *Plugin) Initialize(c config.DeviceConfig) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 防止重复初始化时泄漏旧的 listener 和活跃连接。
	_ = p.destroyLocked()

	p.config = c
	p.connPool = make(map[string]*connector)

	// 初始化连接池
	if err := p.initConnPool(); err != nil {
		// 初始化失败时释放已经启动的连接，避免留下半初始化状态。
		_ = p.destroyLocked()
		driverbox.Log().Error("initialize tcpserver plugin failed", zap.Error(err))
	}
}

// Connector 获取设备连接器
func (p *Plugin) Connector(deviceId string) (connector plugin.Connector, err error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// 获取设备配置
	device, ok := driverbox.CoreCache().GetDevice(deviceId)
	if !ok {
		return nil, errors.New("not found device connection key")
	}

	// 获取连接器
	c, ok := p.connPool[device.ConnectionKey]
	if !ok {
		return nil, errors.New("not found connection key, key is " + device.ConnectionKey)
	}

	return c, nil
}

// Destroy 销毁插件
func (p *Plugin) Destroy() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.destroyLocked()
}

func (p *Plugin) destroyLocked() error {
	var errs []error
	for _, c := range p.connPool {
		if c != nil {
			if err := c.Release(); err != nil {
				errs = append(errs, err)
			}
		}
	}
	p.connPool = nil
	return errors.Join(errs...)
}

// enableConfig 用于区分 enable 字段是否配置。BaseConnection 的 bool 字段无法表达“未配置，默认启用”。
type enableConfig struct {
	Enable *bool `json:"enable"`
}

func connectionEnabled(raw any) (bool, error) {
	var c enableConfig
	if err := convutil.Struct(raw, &c); err != nil {
		return false, err
	}
	if c.Enable == nil {
		return true, nil
	}
	return *c.Enable, nil
}

func normalizeConnectionConfig(c *connectorConfig) {
	if c.BuffSize == 0 {
		c.BuffSize = 1024
	}
	if c.ReadTimeout <= 0 {
		c.ReadTimeout = 30
	}
}

// initConnPool 初始化连接池
func (p *Plugin) initConnPool() (err error) {
	for key, raw := range p.config.Connections {
		var c connectorConfig
		if err = convutil.Struct(raw, &c); err != nil {
			return err
		}
		enabled, err := connectionEnabled(raw)
		if err != nil {
			return err
		}
		if !enabled {
			continue
		}

		c.ConnectionKey = key
		normalizeConnectionConfig(&c)

		conn := newConnector(c)
		if err = conn.startServer(); err != nil {
			return err
		}
		p.connPool[key] = conn
	}
	return nil
}
