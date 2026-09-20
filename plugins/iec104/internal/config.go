package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ibuilding-x/driver-box/v2/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/v2/pkg/config"
	protocol "github.com/ibuilding-x/driver-box/v2/pkg/iec104"
	"github.com/orglibs/go-iecp5/asdu"
)

// connectionConfig 对应 connections 下的一条主站连接。一个 TCP 会话可以承载
// 多个设备、多个公共地址；设备身份与连接身份分开维护，不能用连接 Key 代替设备 ID。
type connectionConfig struct {
	// BaseConnection 提供 enable 和连接 Key；此插件不实现 discover、virtual 或 protocolKey 脚本。
	plugin.BaseConnection
	// Address 为远端从站地址，接受 host[:port] 或 tcp://host[:port]，默认端口 2404。
	Address string `json:"address"`
	// ProtocolLogEnabled 控制本连接的协议报文日志，默认 false，与 DLT645 使用相同字段名。
	// 开启后以 Info 级记录收发 APDU 十六进制和协议摘要；共享连接的所有设备一起生效。
	// 不修改全局日志级别；配置重载后生效，普通运行错误不受此开关影响。
	ProtocolLogEnabled bool `json:"protocolLogEnabled"`
	// CommonAddress 是设备未指定 properties.commonAddress 时使用的 CA，默认 1。
	// 有效范围 1..65534；不使用 0 或广播地址 65535，以免命令扩散到其他站。
	CommonAddress uint16 `json:"commonAddress"`
	// OriginatorAddress 是主站源发地址 OA，范围 0..255，默认 0；与远端约定一致。
	OriginatorAddress uint8 `json:"originatorAddress"`
	// TimeZone 用于解释 CP24/CP56 时间标签和编码对时命令，默认 UTC，可填 Asia/Shanghai。
	TimeZone string `json:"timeZone"`
	// ConnectTimeout 为建立 TCP 会话的 t0，默认 10s，有效范围 1s..255s。
	ConnectTimeout string `json:"connectTimeout"`
	// ReconnectInterval 为拨号失败后的重试间隔，默认 5s，必须大于零。
	// 已连接会话断开后的首次重连仍遵循 fork 的 500ms..1s 随机退避。
	ReconnectInterval string `json:"reconnectInterval"`
	// InterrogationInterval 为每个 CA 的周期总召间隔，默认 5m；0s 关闭周期总召。
	// 即使设为 0s，启动/重连并完成 STARTDT 后仍对每个 CA 总召一次。
	InterrogationInterval string `json:"interrogationInterval"`
	// ClockSyncInterval 为每个 CA 的对时间隔；默认 0s 关闭。非零时每次连接也立即对时。
	ClockSyncInterval string `json:"clockSyncInterval"`
	// CommandTimeout 为每个选择或执行阶段等待 ACT_CON 的上限，默认 10s。
	// 超时表示执行结果未知；插件断开本次会话，不自动重发控制命令。
	CommandTimeout string `json:"commandTimeout"`
}

// settings 是经过校验的运行时配置；初始化后只读，供连接和回调安全共享。
type settings struct {
	connectionConfig
	// params 固定使用 IEC104 宽地址：COT 2 字节、CA 2 字节、IOA 3 字节。
	params asdu.Params
	// 以下时长由同名 JSON 字段解析而来；使用 Duration 避免在定时循环内重复解析字符串。
	connectTimeout        time.Duration
	reconnectInterval     time.Duration
	interrogationInterval time.Duration
	clockSyncInterval     time.Duration
	commandTimeout        time.Duration
	// dialContext 是可选传输工厂；生产使用默认 TCP，测试注入 net.Pipe 跑真实协议状态机。
	dialContext func(context.Context, *url.URL) (net.Conn, error)
}

// parseSettings 先应用默认值再解码。显式的非法值不会被 fork 悄悄替换成默认值。
func parseSettings(raw any) (settings, error) {
	s := settings{connectionConfig: connectionConfig{CommonAddress: 1, TimeZone: "UTC", ConnectTimeout: "10s", ReconnectInterval: "5s", InterrogationInterval: "5m", ClockSyncInterval: "0s", CommandTimeout: "10s"}, params: *asdu.ParamsWide}
	if err := convert(raw, &s.connectionConfig); err != nil {
		return s, err
	}
	if s.Virtual || s.Discover || s.ProtocolKey != "" {
		return s, fmt.Errorf("IEC104 does not support virtual, discover or protocolKey")
	}
	if s.CommonAddress == 0 || s.CommonAddress == 65535 {
		return s, fmt.Errorf("commonAddress must be in [1, 65534]")
	}
	address := s.Address
	if !strings.Contains(address, "://") {
		address = "tcp://" + address
	}
	u, err := url.Parse(address)
	if err != nil || u.Scheme != "tcp" || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "" {
		return s, fmt.Errorf("address must be tcp://host[:port]")
	}
	port := u.Port()
	if port == "" {
		port = "2404"
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return s, fmt.Errorf("invalid TCP port")
	}
	s.Address = "tcp://" + net.JoinHostPort(u.Hostname(), port)
	s.params.InfoObjTimeZone, err = time.LoadLocation(s.TimeZone)
	if err != nil {
		return s, err
	}
	s.params.OrigAddress = asdu.OriginAddr(s.OriginatorAddress)
	for _, field := range []struct {
		name, value string
		target      *time.Duration
		zero        bool
	}{
		{"connectTimeout", s.ConnectTimeout, &s.connectTimeout, false}, {"reconnectInterval", s.ReconnectInterval, &s.reconnectInterval, false},
		{"interrogationInterval", s.InterrogationInterval, &s.interrogationInterval, true}, {"clockSyncInterval", s.ClockSyncInterval, &s.clockSyncInterval, true}, {"commandTimeout", s.CommandTimeout, &s.commandTimeout, false},
	} {
		d, err := time.ParseDuration(field.value)
		if err != nil || d < 0 || (!field.zero && d == 0) {
			return s, fmt.Errorf("invalid %s duration %q", field.name, field.value)
		}
		*field.target = d
	}
	if s.connectTimeout < time.Second || s.connectTimeout > 255*time.Second {
		return s, fmt.Errorf("connectTimeout must be in [1s, 255s]")
	}
	return s, nil
}

func convert(v, target any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, target)
}

// node 是模型点位针对某一设备展开后的配置，地址已经加过设备偏移。
type node struct {
	protocol.Point
	// deviceID 是上报/写入归属的实际设备 ID，不是连接 Key。
	deviceID string
	// name 保留物模型点名；同型号多个设备可以使用相同点名。
	name string
	// access 对应通用 readWrite 字段；同时用于构建上报路由和校验下行操作。
	access config.ReadWrite
}

// route 的作用域为一条连接；同 CA/IOA 的不同值类型仍可独立寻址。
type route struct {
	protocol.Address
	// family 将同一信息类型的无时标、CP24、CP56 变体归并，兼容总召与自发上报。
	family asdu.TypeID
}

// resolvePoint 使用有符号 64 位中间值计算偏移，再校验 24 位范围，防止溢出回绕。
// 设备 properties（字符串值）：
//
//	commonAddress：覆盖连接默认 CA；
//	signalIoaOffset：遥信点的监视偏移，默认 0；
//	telemetryIoaOffset：遥测点的监视偏移，默认 0；
//	commandIoaOffset：遥控偏移；setpointIoaOffset：遥调偏移；均默认 0。
//	控制基础地址取 ext.commandIoa，未配置时取原始 ext.ioa，不继承监视偏移。
//
// CA 始终属于设备，模型 ext 中的 commonAddress 不参与覆盖，以确保模型能跨设备复用。
func resolvePoint(point config.Point, device config.Device, defaultCA uint16) (node, error) {
	n := node{deviceID: device.ID}
	var ok bool
	n.name, ok = point["name"].(string)
	if !ok || n.name == "" {
		return n, fmt.Errorf("missing point name")
	}
	rw, _ := point["readWrite"].(string)
	n.access = config.ReadWrite(rw)
	if n.access != config.ReadWrite_R && n.access != config.ReadWrite_RW && n.access != config.ReadWrite_W {
		return n, fmt.Errorf("point %s requires readWrite R, W or RW", n.name)
	}
	ext, exists := point["ext"]
	if !exists || ext == nil {
		return n, fmt.Errorf("point %s requires IEC104 ext", n.name)
	}
	var fields map[string]json.RawMessage
	if err := convert(ext, &fields); err != nil {
		return n, err
	}
	if _, ok := fields["ioa"]; !ok {
		return n, fmt.Errorf("point %s requires ext.ioa", n.name)
	}
	if err := convert(ext, &n.Point); err != nil {
		return n, err
	}
	ca := int64(defaultCA)
	if v := device.Properties["commonAddress"]; v != "" {
		var err error
		ca, err = parseInteger(v)
		if err != nil {
			return n, fmt.Errorf("invalid device commonAddress: %w", err)
		}
	}
	if ca < 1 || ca > 65534 {
		return n, fmt.Errorf("device commonAddress must be in [1, 65534]")
	}
	n.CommonAddress = uint16(ca)
	// 原始模型地址同样要合法，不能用负偏移掩盖大于 24 位的错误点表。
	if err := n.Validate(); err != nil {
		return n, err
	}
	offsets, err := parseDeviceOffsets(device.Properties)
	if err != nil {
		return n, err
	}
	// 分类只取决于模型的监视 typeId，与值的 Go 类型、readWrite、控制类型无关。
	// 同族 CP24/CP56 变体使用同一偏移，避免总召和自发上报落入不同地址区。
	offset := offsets.telemetry
	switch protocol.MonitoringFamily(asdu.TypeID(n.TypeID)) {
	case asdu.M_SP_NA_1, asdu.M_DP_NA_1, asdu.M_BO_NA_1:
		offset = offsets.signal
	}
	baseCommand := n.IOA
	if n.CommandIOA != nil {
		baseCommand = *n.CommandIOA
	}
	readAddr := int64(n.IOA) + offset
	// 单/双命令是遥控，归一化/标度/短浮点设点是遥调；按 commandType 选取控制区。
	// 不能根据监视 typeId 推断，一个监视点可以配置独立类型的控制命令。
	commandOffset := offsets.command
	switch asdu.TypeID(n.CommandType) {
	case asdu.C_SE_NA_1, asdu.C_SE_NB_1, asdu.C_SE_NC_1:
		commandOffset = offsets.setpoint
	}
	writeAddr := int64(baseCommand) + commandOffset
	if readAddr < 0 || readAddr > 0xffffff || (n.CommandType != 0 && (writeAddr < 0 || writeAddr > 0xffffff)) {
		return n, fmt.Errorf("resolved IOA outside [0, 16777215]")
	}
	n.IOA = uint32(readAddr)
	if n.CommandType != 0 {
		value := uint32(writeAddr)
		n.CommandIOA = &value
	}
	if err := n.Validate(); err != nil {
		return n, err
	}
	if n.access != config.ReadWrite_R && n.CommandType == 0 {
		return n, fmt.Errorf("writable point %s requires commandType", n.name)
	}
	return n, nil
}

// deviceOffsets 保存已校验的设备地址偏移，四类分别从各自的模型基础地址计算。
// 不设置时为 0；四者互不继承，也不相加。全量解析可尽早发现尚未使用的错误配置。
type deviceOffsets struct {
	// signal 用于单点、双点遥信及表示状态集合的 32 位位串（含带时标变体）。
	signal int64
	// telemetry 用于归一化、标度、短浮点测量值；数值型步位置和累计量也使用此区。
	// 这是本插件的地址分区约定，不表示 IEC104 协议强制这些类型共享地址区。
	telemetry int64
	// command 仅用于单命令/双命令遥控（45/46），基于 commandIoa 或原始模型 ioa。
	command int64
	// setpoint 仅用于归一化/标度/短浮点遥调（48/49/50），不继承遥控或遥测偏移。
	setpoint int64
}

func parseDeviceOffsets(properties map[string]string) (deviceOffsets, error) {
	var offsets deviceOffsets
	if _, exists := properties["ioaOffset"]; exists {
		return offsets, fmt.Errorf("ioaOffset is no longer supported; use signalIoaOffset, telemetryIoaOffset, commandIoaOffset and setpointIoaOffset")
	}
	for _, field := range []struct {
		name  string // properties 中的设备属性名；属性值必须是整数字符串。
		value *int64 // 对应运行时偏移；未设置的属性保留零值。
	}{
		{"signalIoaOffset", &offsets.signal},
		{"telemetryIoaOffset", &offsets.telemetry},
		{"commandIoaOffset", &offsets.command},
		{"setpointIoaOffset", &offsets.setpoint},
	} {
		raw, exists := properties[field.name]
		if !exists {
			continue
		}
		value, err := parseInteger(raw)
		if err != nil {
			return offsets, fmt.Errorf("invalid %s: %w", field.name, err)
		}
		// 相加前限制偏移自身范围，防止极端输入使 int64 溢出；最终 IOA 仍需单独校验。
		if value < -0xffffff || value > 0xffffff {
			return offsets, fmt.Errorf("%s must be in [-16777215, 16777215]", field.name)
		}
		*field.value = value
	}
	return offsets, nil
}

func parseInteger(s string) (int64, error) {
	base := 10
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "-0x") || strings.HasPrefix(s, "0X") {
		base = 0
	}
	return strconv.ParseInt(s, base, 64)
}
