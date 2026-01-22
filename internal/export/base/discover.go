package base

import (
	"encoding/json"
	"net"
	"os"
	"strings"

	"github.com/ibuilding-x/driver-box/internal/core"
	"github.com/ibuilding-x/driver-box/internal/logger"
	"github.com/ibuilding-x/driver-box/pkg/config"
	"github.com/ibuilding-x/driver-box/pkg/convutil"
	"go.uber.org/zap"
)

func (export *Export) udpDiscover() {
	// 监听UDP端口
	port := 9090
	httpListen := os.Getenv(config.ENV_UDP_DISCOVER_LISTEN)
	if httpListen != "" {
		p, e := convutil.Int64(httpListen)
		if e != nil {
			logger.Logger.Error("udp discover listen port error", zap.String(config.ENV_UDP_DISCOVER_LISTEN, httpListen), zap.Error(e))
		} else {
			port = int(p)
		}
	}
	addr := net.UDPAddr{
		Port: port,
		IP:   net.IPv4(0, 0, 0, 0),
	}
	var err error
	export.discoverConn, err = net.ListenUDP("udp", &addr)
	if err != nil {
		logger.Logger.Error("UDP监听失败", zap.Error(err))
		return
	}

	logger.Logger.Info("UDP服务已启动.", zap.Int("port", port))

	buffer := make([]byte, 1024)
	for export.discoverEnable {
		n, remoteAddr, err := export.discoverConn.ReadFromUDP(buffer)
		if err != nil {
			logger.Logger.Error("读取UDP数据失败", zap.Error(err))
			continue
		}

		data := string(buffer[:n])
		logger.Logger.Info("收到UDP数据", zap.String("data", data), zap.String("remoteAddr", remoteAddr.String()))

		// 基础验证
		if !validateRequest(data) {
			logger.Logger.Error("验证失败", zap.String("data", data))
			continue
		}

		type Resp struct {
			config.Metadata
			Port string `json:"port"`
		}
		resp := Resp{
			Metadata: core.Metadata,
			Port:     export.httpListen,
		}
		// 获取网关Metadata信息
		response, err := json.Marshal(resp)
		if err != nil {
			logger.Logger.Error("JSON编码失败", zap.Error(err))
			continue
		}

		// 返回响应
		_, err = export.discoverConn.WriteToUDP(response, remoteAddr)
		if err != nil {
			logger.Logger.Error("发送响应失败", zap.Error(err))
		}
	}
}

func (export *Export) stopDiscover() {
	export.discoverEnable = false
	export.discoverConn.Close()
	logger.Logger.Info("UDP服务已停止.")
}

// 验证请求数据
func validateRequest(data string) bool {
	// 简单验证示例：检查数据是否包含特定令牌
	return strings.Contains(data, "driver-box")
}
