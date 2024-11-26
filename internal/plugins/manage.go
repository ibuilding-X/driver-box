// 插件管理器

package plugins

import (
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"sync"
)

// Manager 插件管理器
var Manager *manager

func init() {
	Manager = &manager{
		plugins: &sync.Map{},
	}
}

// manager 管理器
type manager struct {
	plugins *sync.Map
}

// 注册自定义插件
func (m *manager) Register(name string, plugin plugin.Plugin) error {
	if _, ok := m.plugins.Load(name); ok {
		fmt.Printf("plugin %s already exists, replace it", name)
	}
	fmt.Printf("register plugin: %s\n", name)
	m.plugins.Store(name, plugin)
	return nil
}

// Get 获取插件实例
func (m *manager) Get(c config.Config) (p plugin.Plugin, err error) {
	if raw, ok := m.plugins.Load(c.ProtocolName); ok {
		p = raw.(plugin.Plugin)
	} else {
		err = fmt.Errorf("plugin:[%s] not found", c.ProtocolName)
	}
	return
}
