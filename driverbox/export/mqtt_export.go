package export

import (
	"crypto/tls"
	"encoding/json"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"go.uber.org/zap"
	"log"
	"time"
)

type MqttExport struct {
	Broker      string `json:"broker"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	ClientID    string `json:"client_id"`
	init        bool
	client      mqtt.Client
	handler     mqtt.MessageHandler
	ExportTopic string
}

func (export *MqttExport) Init() error {
	if len(export.ExportTopic) == 0 {
		panic("exportTopic is blank")
	}
	options := mqtt.NewClientOptions()
	options.AddBroker(export.Broker)
	options.SetUsername(export.Username)
	options.SetPassword(export.Password)
	options.SetClientID(export.ClientID)
	// tsl 设置
	if options.Servers[0].Scheme == "ssl" {
		options.SetTLSConfig(&tls.Config{
			InsecureSkipVerify: true,
		})
	}
	options.SetOnConnectHandler(export.onConnectHandler)
	options.SetConnectionLostHandler(export.onConnectionLostHandler)
	export.client = mqtt.NewClient(options)
	token := export.client.Connect()
	if token.WaitTimeout(5*time.Second) && token.Error() != nil {
		return token.Error()
	}
	return nil
}

// onConnectHandler 连接成功
func (export *MqttExport) onConnectHandler(client mqtt.Client) {
	log.Println("mqttExport init success")
	export.init = true
}

// onConnectionLostHandler 连接丢失
func (export *MqttExport) onConnectionLostHandler(client mqtt.Client, err error) {
	log.Fatal("local mqtt connect lost", zap.Error(err))
}

// ExportTo 导出消息：写入Edgex总线、MQTT上云
func (export *MqttExport) ExportTo(deviceData plugin.DeviceData) {
	log.Println("export...")
	bytes, _ := json.Marshal(deviceData)
	token := export.client.Publish(export.ExportTopic, 0, false, bytes)
	if token.Error() != nil {
		log.Fatal(token.Error())
	}
}

func (export *MqttExport) SendStatusChangeNotification(deviceName string, online bool) error {
	export.client.Publish("/driverbox/event/"+export.ClientID, 0, false, map[string]any{"deviceName": deviceName, "online": online})
	return nil
}

func (export *MqttExport) IsReady() bool {
	return export.init
}
