package internal

import (
	"fmt"
	"sync"

	"github.com/ibuilding-x/driver-box/v2/driverbox"
	"github.com/ibuilding-x/driver-box/v2/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/v2/pkg/config"
	"go.uber.org/zap"
)

// ProtocolName 同时对应插件注册名和 res/driver/iec104 配置目录。
const ProtocolName = "iec104"

// Plugin 为所有 IEC104 主站连接提供生命周期管理和按设备寻址。
type Plugin struct {
	mu          sync.RWMutex          // 保护连接表，允许业务查询与销毁并发。
	connections map[string]*connector // 连接 Key -> 唯一 TCP 会话。
	devices     map[string]*connector // 设备 ID -> 所属连接；允许多对一。
}

var _ plugin.Plugin = (*Plugin)(nil)

func (p *Plugin) Initialize(cfg config.DeviceConfig) {
	_ = p.Destroy()
	p.mu.Lock()
	defer p.mu.Unlock()
	p.connections = make(map[string]*connector)
	p.devices = make(map[string]*connector)
	for key, raw := range cfg.Connections {
		var base plugin.BaseConnection
		if err := convert(raw, &base); err != nil {
			driverbox.Log().Error("invalid IEC104 connection", zap.String("connection", key), zap.Error(err))
			continue
		}
		if !base.Enable {
			continue
		}
		s, err := parseSettings(raw)
		if err == nil {
			s.ConnectionKey = key
		}
		var c *connector
		if err == nil {
			c, err = newConnector(s, cfg, driverbox.Export)
		}
		if err == nil {
			for device := range c.nodes {
				if _, exists := p.devices[device]; exists {
					err = fmt.Errorf("duplicate device ID %s", device)
					break
				}
			}
		}
		if err != nil {
			driverbox.Log().Error("initialize IEC104 connection", zap.String("connection", key), zap.Error(err))
			continue
		}
		if len(c.nodes) == 0 {
			continue
		}
		if err := c.start(); err != nil {
			driverbox.Log().Error("start IEC104 connection", zap.String("connection", key), zap.Error(err))
			continue
		}
		p.connections[key] = c
		for device := range c.nodes {
			p.devices[device] = c
		}
	}
}

func (p *Plugin) Connector(deviceID string) (plugin.Connector, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	c, ok := p.devices[deviceID]
	if !ok {
		return nil, fmt.Errorf("IEC104 device %q has no enabled connection", deviceID)
	}
	return c, nil
}

func (p *Plugin) Destroy() error {
	p.mu.Lock()
	connections := p.connections
	p.connections = nil
	p.devices = nil
	p.mu.Unlock()
	for _, c := range connections {
		_ = c.Close()
	}
	return nil
}
