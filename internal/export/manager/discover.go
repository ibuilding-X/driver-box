package manager

import (
	"log"
	"net"
)

func udpDiscover() {
	addr := net.UDPAddr{
		Port: 8888,
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

		log.Printf("收到来自 %v 的数据: %s", remoteAddr, string(buffer[:n]))
	}
}
