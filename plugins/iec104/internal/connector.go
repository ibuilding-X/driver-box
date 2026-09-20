package internal

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/ibuilding-x/driver-box/v2/driverbox"
	"github.com/ibuilding-x/driver-box/v2/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/v2/pkg/config"
	protocol "github.com/ibuilding-x/driver-box/v2/pkg/iec104"
	"github.com/orglibs/go-iecp5/asdu"
	"github.com/orglibs/go-iecp5/cs104"
)

// masterClient 隔离传输与业务映射；默认实现为 circutor fork 的 cs104.Client。
type masterClient interface {
	asdu.Connect
	Start() error
	Close() error
	Wait()
	Disconnect()
	IsConnected() bool
}

// pendingCommand 是当前唯一在等待确认的控制阶段，由 mu 保护。
type pendingCommand struct {
	// frame 保存原始请求，用 CA、OA、类型、地址、值、选择位匹配响应。
	frame *asdu.ASDU
	// result 容量为 1，回调只投递结果，不等待调用方消费，避免堵塞收包状态机。
	result chan error
}

// telemetryBatch carries its source session so queued samples cannot revive a
// device after that session has disconnected or been replaced.
type telemetryBatch struct {
	generation uint64
	values     []plugin.DeviceData
}

// connector 管理一条 TCP 主站会话。所有地址表初始化后只读，生命周期状态由锁保护。
type connector struct {
	// settings 是连接参数的不可变快照。
	settings settings
	// client 负责 TCP、STARTDT、I/S/U 帧、序号、链路确认和自动重连。
	client masterClient
	// nodes 的两级 Key 分别是设备 ID、点名，确保同模型设备不会覆盖彼此。
	nodes map[string]map[string]node
	// routes 将实际 CA/IOA 唯一映射到业务设备和点名，不依赖报文的具体编码类型。
	routes map[protocol.Address]node
	// stations 是去重且排序的 CA；总召和对时按 CA 发起，而非按设备重复发起。
	stations []uint16
	// export 接入 driverbox.Export；测试可注入收集器，不启动整个应用。
	export func([]plugin.DeviceData)
	// exportMu orders telemetry publication and device-offline transitions.
	exportMu    sync.Mutex
	markOffline func(string)
	// ctx/cancel 统一取消维护任务、上报任务、确认等待和正在入队的上报。
	ctx    context.Context
	cancel context.CancelFunc
	// done 在维护任务退出时关闭，exportDone 在上报任务退出时关闭。
	done       chan struct{}
	exportDone chan struct{}
	// closeOnce 保证热重载、重复 Destroy、并发 Close 只关闭一次。
	closeOnce sync.Once
	// sendMu 将整批读写串行化，选择与执行之间不插入其他控制操作。
	sendMu sync.Mutex
	// mu 保护 pending、generation 和超时封锁状态；等待确认时不得持有此锁。
	mu      sync.Mutex
	pending *pendingCommand
	// generation 每次 TCP 成功连接都递增，能识别两次调度之间发生的快速重连。
	generation uint64
	// blockedGeneration 标记结果未知的旧会话；重连后新 generation 自动解除封锁。
	blockedGeneration uint64
	blocked           bool
	// telemetry 将上报与协议回调分离，避免 Export 中的控制操作堵住其自身 ACT_CON。
	// 队列容量有限；满时施加背压，关闭时通过 ctx 解除等待。
	telemetry chan telemetryBatch
}

var _ plugin.Connector = (*connector)(nil)
var _ cs104.ClientHandlerInterface = (*connector)(nil)

// newConnector 先完整展开并校验本连接所有设备；任一地址冲突会拒绝整条连接，
// 不创建部分可用的地址表。此函数只构建配置，start 才开始通信。
func newConnector(s settings, cfg config.DeviceConfig, export func([]plugin.DeviceData)) (result *connector, err error) {
	ctx, cancel := context.WithCancel(context.Background())
	// 任何配置错误都释放本次构建的上下文；成功后生命周期交给 connector.Close。
	defer func() {
		if err != nil {
			cancel()
		}
	}()
	c := &connector{settings: s, nodes: make(map[string]map[string]node), routes: make(map[protocol.Address]node), export: export, ctx: ctx, cancel: cancel, done: make(chan struct{}), telemetry: make(chan telemetryBatch, 256), exportDone: make(chan struct{})}
	c.markOffline = func(id string) { _ = driverbox.Shadow().SetOffline(id) }
	stations := make(map[uint16]bool)
	commands := make(map[commandRoute]node)
	for _, model := range cfg.DeviceModels {
		for _, device := range model.Devices {
			if device.ConnectionKey != s.ConnectionKey {
				continue
			}
			if device.ID == "" {
				return nil, fmt.Errorf("empty device ID")
			}
			if _, ok := c.nodes[device.ID]; ok {
				return nil, fmt.Errorf("duplicate device %s", device.ID)
			}
			points := make(map[string]node)
			for _, point := range model.DevicePoints {
				n, err := resolvePoint(point, device, s.CommonAddress)
				if err != nil {
					cancel()
					return nil, fmt.Errorf("device %s: %w", device.ID, err)
				}
				if _, ok := points[n.name]; ok {
					return nil, fmt.Errorf("duplicate point %s/%s", device.ID, n.name)
				}
				key := n.Address
				if n.access != config.ReadWrite_W {
					if previous, ok := c.routes[key]; ok {
						return nil, fmt.Errorf("monitoring address conflict CA=%d IOA=%d: %s/%s and %s/%s", n.CommonAddress, n.IOA, previous.deviceID, previous.name, n.deviceID, n.name)
					}
					c.routes[key] = n
				}
				if n.access != config.ReadWrite_R {
					key := commandRoute{n.WriteAddress(), asdu.TypeID(n.CommandType)}
					if previous, ok := commands[key]; ok {
						return nil, fmt.Errorf("command address conflict: %s/%s and %s/%s", previous.deviceID, previous.name, n.deviceID, n.name)
					}
					commands[key] = n
				}
				points[n.name] = n
				stations[n.CommonAddress] = true
			}
			if len(points) > 0 {
				c.nodes[device.ID] = points
			}
		}
	}
	for ca := range stations {
		c.stations = append(c.stations, ca)
	}
	sort.Slice(c.stations, func(i, j int) bool { return c.stations[i] < c.stations[j] })
	option := cs104.NewOption().SetParams(&s.params).SetAutoReconnect(true).SetReconnectInterval(s.reconnectInterval)
	option.DialContext = s.dialContext
	option.OnAPDU = newProtocolTrace(s, driverbox.Log())
	link := cs104.DefaultConfig()
	link.ConnectTimeout0 = s.connectTimeout
	option.SetConfig(link)
	if err := option.AddRemoteServer(s.Address); err != nil {
		cancel()
		return nil, err
	}
	client := cs104.NewClient(c, option)
	client.SetOnConnectHandler(func(client *cs104.Client) {
		c.mu.Lock()
		c.generation++
		c.mu.Unlock()
		client.SendStartDt()
	})
	client.SetConnectionLostHandler(func(*cs104.Client) { c.connectionLost() })
	c.client = client
	return c, nil
}

func (c *connector) start() error {
	if err := c.client.Start(); err != nil {
		c.cancel()
		return err
	}
	go c.maintain()
	go func() {
		defer close(c.exportDone)
		for {
			select {
			case <-c.ctx.Done():
				return
			case data := <-c.telemetry:
				c.exportBatch(data)
			}
		}
	}()
	return nil
}

// maintain 等待 STARTDT 确认后，按去重 CA 发起首次/周期总召和可选对时。
// 只有成功入发送队列才推进时间戳；队列满时下一轮重试。总召不重试控制指令。
func (c *connector) maintain() {
	defer close(c.done)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	var generation uint64
	gi := make(map[uint16]time.Time)
	clocks := make(map[uint16]time.Time)
	for {
		select {
		case <-c.ctx.Done():
			return
		case now := <-ticker.C:
			if !c.client.IsActive() {
				continue
			}
			c.mu.Lock()
			current := c.generation
			c.mu.Unlock()
			if current != generation {
				generation = current
				clear(gi)
				clear(clocks)
			}
			for _, ca := range c.stations {
				if gi[ca].IsZero() || (c.settings.interrogationInterval > 0 && now.Sub(gi[ca]) >= c.settings.interrogationInterval) {
					if err := asdu.InterrogationCmd(c.wire(), asdu.CauseOfTransmission{Cause: asdu.Activation}, asdu.CommonAddr(ca), asdu.QOIStation); err == nil {
						gi[ca] = now
					}
				}
				if c.settings.clockSyncInterval > 0 && (clocks[ca].IsZero() || now.Sub(clocks[ca]) >= c.settings.clockSyncInterval) {
					if err := asdu.ClockSynchronizationCmd(c.wire(), asdu.CauseOfTransmission{Cause: asdu.Activation}, asdu.CommonAddr(ca), true, now); err == nil {
						clocks[ca] = now
					}
				}
			}
		}
	}
}

// wireConnection 为 fork 生成的总召/对时报文补齐配置的源发地址 OA。
type wireConnection struct {
	asdu.Connect
	// origin 是本主站的 OA，不是远端的 CA。
	origin asdu.OriginAddr
}

func (w wireConnection) Send(a *asdu.ASDU) error { a.OrigAddr = w.origin; return w.Connect.Send(a) }
func (c *connector) wire() wireConnection {
	return wireConnection{c.client, c.settings.params.OrigAddress}
}

// Release pending commands before waiting for an exporter: an exporter may be
// waiting for a command confirmation from the same connection.
func (c *connector) connectionLost() {
	c.failPending(errors.New("IEC104 connection lost"))
	c.exportMu.Lock()
	defer c.exportMu.Unlock()
	c.mu.Lock()
	c.generation++ // Invalidate all samples queued before this disconnect.
	c.mu.Unlock()
	for id := range c.nodes {
		c.markOffline(id)
	}
}

func (c *connector) exportBatch(data telemetryBatch) {
	c.exportMu.Lock()
	defer c.exportMu.Unlock()
	c.mu.Lock()
	current := c.generation
	c.mu.Unlock()
	if c.ctx.Err() == nil && data.generation == current && c.client.IsConnected() {
		c.export(data.values)
	}
}
