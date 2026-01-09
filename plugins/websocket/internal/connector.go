package internal

import (
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/ibuilding-x/driver-box/driverbox"
	"github.com/ibuilding-x/driver-box/driverbox/common"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/library"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type connectorConfig struct {
	plugin.BaseConnection
	Host string `json:"host"`
	Port string `json:"port"`
	//匹配路径
	Pattern string `json:"pattern"`
}

// 消息解码结构体
type decodeStruct struct {
	Path       string            `json:"path"`       // 请求路径
	Method     string            `json:"method"`     // 请求方法
	Header     map[string]string `json:"header"`     //请求头
	RemoteAddr string            `json:"remoteAddr"` //客户端地址
	Event      string            `json:"event"`      //事件类型：connect、read、close
	Payload    string            `json:"payload"`    //读取到的消息体
}

// 消息编码结构体
type encodeStruct struct {
	connection *websocket.Conn
	payload    string
}

type connector struct {
	config connectorConfig
	server *http.Server
	//设备与连接的映射
	deviceMappingConn *sync.Map
	//连接与设备的映射
	connMappingDevice *sync.Map
}

// Release 释放资源
func (c *connector) Release() (err error) {
	return nil
}

// Send 被动接收数据模式，无需实现
func (c *connector) Send(raw interface{}) (err error) {
	data := raw.(encodeStruct)
	return data.connection.WriteMessage(websocket.TextMessage, []byte(data.payload))
}

// startServer 启动服务
func (c *connector) startServer() {
	if !c.config.Enable {
		helper.Logger.Warn("websocket connector is not enable", zap.Any("connector", c.config))
		return
	}
	//复用driver-box自身服务
	if c.config.Port == helper.EnvConfig.HttpListen {
		helper.Logger.Error("websocket connector port is same as driver-box http listen port", zap.Any("connector", c.config))
		return
	}
	//启动新的服务
	serverMux := &http.ServeMux{}
	c.server = &http.Server{
		Addr:    ":" + c.config.Port,
		Handler: serverMux,
	}
	c.handleFunc(serverMux)
	go func() {
		e := c.server.ListenAndServe()
		if e != nil {
			helper.Logger.Error("websocket connector start server error", zap.Any("error", e))
		}
	}()
}

func (c *connector) handleFunc(server *http.ServeMux) {
	server.HandleFunc(c.config.Pattern, func(writer http.ResponseWriter, request *http.Request) {
		conn, err := upgrader.Upgrade(writer, request, nil)
		if err != nil {
			fmt.Println("Failed to upgrade connection:", err)
			return
		}
		defer conn.Close()
		header := make(map[string]string)
		for k, v := range request.Header {
			header[k] = v[0]
		}
		decode := decodeStruct{
			Path:       request.URL.Path,
			Method:     request.Method,
			Header:     header,
			RemoteAddr: request.RemoteAddr,
			Event:      "connect",
		}
		for {
			messageType, p, err := conn.ReadMessage()
			if err != nil {
				fmt.Println("Failed to read message:", err)
				break
			}
			helper.Logger.Info("Received message", zap.Any("messageType", messageType), zap.Any("payload", string(p)))
			decode.Event = "read"
			decode.Payload = string(p)
			deviceDatas, err := library.Protocol().Decode(c.config.ProtocolKey, decode)
			if err != nil {
				fmt.Println("Failed to decode message:", err)
				continue
			}

			//更新设备与连接的映射关系
			for _, deviceData := range deviceDatas {
				//更新映射关系
				preConn, ok := c.deviceMappingConn.Swap(deviceData.ID, conn)
				if preConn == conn {
					continue
				}
				//在新连接中加入当前设备
				devices, ok := c.connMappingDevice.Load(conn)
				if ok {
					devices = append(devices.([]string), deviceData.ID)
				} else {
					devices = []string{deviceData.ID}
				}
				c.connMappingDevice.Store(conn, devices)
			}
			//自动添加设备
			common.WrapperDiscoverEvent(deviceDatas, c.config.ConnectionKey, ProtocolName)
			driverbox.Export(deviceDatas)
		}

		//移除映射
		devices, ok := c.connMappingDevice.LoadAndDelete(conn)
		if ok {
			for _, device := range devices.([]string) {
				//若移除失败，说明当前设备最近一次是通过其他 TCP 连接上报的，则无需处理
				//否则，将该设备设置为：离线
				deleted := c.deviceMappingConn.CompareAndDelete(device, conn)
				if deleted {
					_ = helper.DeviceShadow.SetOffline(device)
				}
			}
		}
	})
}

// Encode 编码数据，无需实现
func (c *connector) Encode(deviceId string, mode plugin.EncodeMode, values ...plugin.PointData) (res interface{}, err error) {
	payload, err := library.Protocol().Encode(c.config.ProtocolKey, library.ProtocolEncodeRequest{
		DeviceId: deviceId,
		Mode:     mode,
		Points:   values,
	})
	if err != nil {
		return nil, err
	}
	conn, ok := c.deviceMappingConn.Load(deviceId)
	if !ok {
		return nil, errors.New("device is disconnected")
	}
	return encodeStruct{
		payload:    payload,
		connection: conn.(*websocket.Conn),
	}, nil
}

// Decode 解码数据，调用动态脚本解析
func (a *connector) Decode(raw interface{}) (res []plugin.DeviceData, err error) {
	return nil, common.NotSupportDecode
}
