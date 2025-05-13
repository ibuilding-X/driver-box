package basic

import (
	"encoding/json"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/helper/utils"
	"github.com/ibuilding-x/driver-box/internal/core"
	"go.uber.org/zap"
	"log"
	"net"
	"os"
	"strings"
)

func udpDiscover() {
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

	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		log.Printf("UDP监听失败: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("UDP服务已启动，监听端口: 8888")

	buffer := make([]byte, 1024)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Printf("读取UDP数据失败: %v", err)
			continue
		}

		data := string(buffer[:n])
		log.Printf("收到来自 %v 的数据: %s", remoteAddr, data)

		// 基础验证
		if !validateRequest(data) {
			log.Printf("验证失败: %s", data)
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
			log.Printf("JSON编码失败: %v", err)
			continue
		}

		// 返回响应
		_, err = conn.WriteToUDP(response, remoteAddr)
		if err != nil {
			log.Printf("发送响应失败: %v", err)
		}
	}
}

// 验证请求数据
func validateRequest(data string) bool {
	// 简单验证示例：检查数据是否包含特定令牌
	return strings.Contains(data, "driver-box")
}
