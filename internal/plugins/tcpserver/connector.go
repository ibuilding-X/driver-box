package tcpserver

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/common"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/driverbox/plugin/callback"
	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"
	"net"
)

type connector struct {
	config    connectorConfig
	plugin    *Plugin
	conn      net.Listener
	scriptDir string // 脚本目录名称
	ls        *lua.LState
}

// connectorConfig 连接器配置
type connectorConfig struct {
	Host     string `json:"host"`
	Port     uint16 `json:"port"`
	BuffSize uint   `json:"buffSize"`
}

func (c *connector) Send(raw interface{}) (err error) {
	return nil
}

func (c *connector) Release() (err error) {
	if c.conn != nil {
		return c.conn.Close()
	}
	return
}

// startServer 启动 TCP 服务
func (c *connector) startServer() (err error) {
	addr := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return
	}
	// 携程启动，防止阻塞
	go func(listener net.Listener, addr string) {
		helper.Logger.Info("Listening and serving TCP", zap.String("addr", addr))
		// 循环接收 TCP Client 连接
		for {
			conn, err := listener.Accept()
			if err != nil {
				helper.Logger.Error("TCP accept connection error", zap.Error(err))
				break
			}
			helper.Logger.Debug("tcp client is connected", zap.String("remoteAddr", conn.RemoteAddr().String()))
			go c.handelConn(conn)
		}
		helper.Logger.Warn("End listening and serving TCP", zap.String("addr", addr))
	}(listener, addr)

	c.conn = listener
	return nil
}

// handelConn 处理 TCP 连接
func (c *connector) handelConn(conn net.Conn) {
	defer conn.Close()

	buf := make([]byte, c.config.BuffSize)
	reader := bufio.NewReader(conn)
	for {
		n, err := reader.Read(buf[:])
		if err != nil {
			c.plugin.logger.Error("tcp connection read error", zap.Error(err))
			break
		}
		data := protoData{Raw: string(buf[:n])}
		// 接收数据，调用回调函数
		if res, err := c.Decode(data.ToJSON()); err != nil {
			c.plugin.logger.Error("tcp_server callback error", zap.Error(err))
		} else {
			callback.ExportTo(res)
		}
	}
}

// protoData 协议数据
type protoData struct {
	Raw string `json:"raw"`
}

// ToJSON 协议数据转 json 字符串
func (pd protoData) ToJSON() string {
	b, _ := json.Marshal(pd)
	return string(b)
}

// Encode 编码
// 暂无实现
func (a *connector) Encode(deviceSn string, mode plugin.EncodeMode, values ...plugin.PointData) (res interface{}, err error) {
	return nil, common.NotSupportEncode
}

// Decode 解码
func (a *connector) Decode(raw interface{}) (res []plugin.DeviceData, err error) {
	return helper.CallLuaConverter(a.ls, "decode", raw)
}
