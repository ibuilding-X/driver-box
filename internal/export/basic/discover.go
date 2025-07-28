package basic

import (
	"encoding/json"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/helper/utils"
	"github.com/ibuilding-x/driver-box/internal/core"
	"go.uber.org/zap"
	"net"
	"os"
	"strings"
)

type Discover struct {
	conn   *net.UDPConn
	enable bool
}

func NewDiscover() *Discover {
	return &Discover{
		enable: true,
	}
}

func (discover *Discover) udpDiscover() {
	// 监听UDP端口
	port := 9090
	httpListen := os.Getenv(config.ENV_UDP_DISCOVER_LISTEN)
	if httpListen != "" {
		p, e := utils.Conv2Int64(httpListen)
		if e != nil {
			helper.Logger.Error("udp discover listen port error", zap.String(config.ENV_UDP_DISCOVER_LISTEN, httpListen), zap.Error(e))
		} else {
			port = int(p)
		}
	}
	addr := net.UDPAddr{
		Port: port,
		IP:   net.IPv4(0, 0, 0, 0),
	}
	var err error
	discover.conn, err = net.ListenUDP("udp", &addr)
	if err != nil {
		helper.Logger.Error("UDP监听失败", zap.Error(err))
		return
	}

	helper.Logger.Info("UDP服务已启动.", zap.Int("port", port))

	buffer := make([]byte, 1024)
	for discover.enable {
		n, remoteAddr, err := discover.conn.ReadFromUDP(buffer)
		if err != nil {
			helper.Logger.Error("读取UDP数据失败", zap.Error(err))
			continue
		}

		data := string(buffer[:n])
		helper.Logger.Info("收到UDP数据", zap.String("data", data), zap.String("remoteAddr", remoteAddr.String()))

		// 基础验证
		if !validateRequest(data) {
			helper.Logger.Error("验证失败", zap.String("data", data))
			continue
		}

		type Resp struct {
			config.Metadata
			Port string `json:"port"`
		}
		resp := Resp{
			Metadata: core.Metadata,
			Port:     helper.EnvConfig.HttpListen,
		}
		// 获取网关Metadata信息
		response, err := json.Marshal(resp)
		if err != nil {
			helper.Logger.Error("JSON编码失败", zap.Error(err))
			continue
		}

		// 返回响应
		_, err = discover.conn.WriteToUDP(response, remoteAddr)
		if err != nil {
			helper.Logger.Error("发送响应失败", zap.Error(err))
		}
	}
}

func (discover *Discover) stopDiscover() {
	discover.enable = false
	discover.conn.Close()
	helper.Logger.Info("UDP服务已停止.")
}

// 验证请求数据
func validateRequest(data string) bool {
	// 简单验证示例：检查数据是否包含特定令牌
	return strings.Contains(data, "driver-box")
}
