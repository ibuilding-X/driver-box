package mqtt

import (
	"encoding/json"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/ibuilding-x/driver-box/driverbox/common"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/driverbox/plugin/callback"
	"github.com/ibuilding-x/driver-box/internal/library"
	"github.com/ibuilding-x/driver-box/internal/logger"
	"go.uber.org/zap"
)

type connector struct {
	plugin *Plugin
	client mqtt.Client
	config ConnectConfig
}

type EncodeData struct {
	Topic   string `json:"topic"`
	Payload string `json:"payload"`
}

// ProtocolAdapter 协议适配器
func (conn *connector) ProtocolAdapter() plugin.ProtocolAdapter {
	return conn
}
func (conn *connector) Send(data interface{}) error {
	res := []byte(data.(string))
	var encodeData EncodeData
	err := json.Unmarshal(res, &encodeData)
	if err != nil {
		conn.plugin.logger.Error(fmt.Sprintf("unmarshal error: %s", err.Error()))
		conn.plugin.logger.Error(fmt.Sprintf("origin data is: %s", data.(string)))
		return err
	}
	if token := conn.client.Publish(encodeData.Topic, 0, false, encodeData.Payload); token.Wait() && token.Error() != nil {
		conn.plugin.logger.Error(fmt.Sprintf("publish %s to topic %s error: %s",
			encodeData.Payload, encodeData.Topic, token.Error().Error()))
		return token.Error()
	}
	return nil
}

func (conn *connector) Release() (err error) {
	return nil
}

type ConnectConfig struct {
	plugin.BaseConnection
	ClientId string   `json:"clientId"`
	Broker   string   `json:"broker"`
	Username string   `json:"username"`
	Password string   `json:"password"`
	Topics   []string `json:"topics"`
}

func (conn *connector) connect(connectConfig ConnectConfig) error {
	options := conn.newMqttClientOptions(connectConfig)
	client := mqtt.NewClient(options)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	conn.client = client
	return nil
}

// newMqttClientOptions 初始化 MQTT Client Options
func (conn *connector) newMqttClientOptions(connectConfig ConnectConfig) *mqtt.ClientOptions {
	opts := mqtt.NewClientOptions()
	opts.SetAutoReconnect(true)
	opts.SetClientID(connectConfig.ClientId)
	opts.AddBroker(connectConfig.Broker)
	opts.SetUsername(connectConfig.Username)
	opts.SetPassword(connectConfig.Password)
	opts.SetOnConnectHandler(conn.onConnectHandler)
	return opts
}

// onConnectHandler 执行连接回调： 完成订阅
func (conn *connector) onConnectHandler(client mqtt.Client) {
	for _, topic := range conn.config.Topics {
		if token := client.Subscribe(topic, 0, conn.onReceiveHandler); token.Wait() && token.Error() != nil {
			conn.plugin.logger.Error(fmt.Sprintf("unable to subscribe topic: %s for client: %s", topic, conn.config.ClientId))
			continue
		}
	}
}

type Msg struct {
	Topic   string `json:"topic"`
	Payload string `json:"payload"`
}

// onReceiveHandler 消息回调
func (conn *connector) onReceiveHandler(_ mqtt.Client, message mqtt.Message) {
	msg := Msg{
		Topic:   message.Topic(),
		Payload: string(message.Payload()),
	}
	// 执行回调 写入消息总线
	deviceData, err := library.Protocol().Decode(conn.config.ProtocolKey, msg)
	if err != nil {
		logger.Logger.Error("decode error", zap.Error(err))
		return
	}
	//自动添加设备
	common.WrapperDiscoverEvent(deviceData, conn.config.ConnectionKey, ProtocolName)
	callback.ExportTo(deviceData)
}
