package driverbox

func RegisterAllPlugins() error {
	if err := RegisterModbusPlugin(); err != nil {
		return err
	}
	if err := RegisterBacnetPlugin(); err != nil {
		return err
	}
	if err := RegisterHttpServerPlugin(); err != nil {
		return err
	}
	if err := RegisterHttpClientPlugin(); err != nil {
		return err
	}
	if err := RegisterWebsocketPlugin(); err != nil {
		return err
	}
	if err := RegisterTcpServerPlugin(); err != nil {
		return err
	}
	if err := RegisterMqttPlugin(); err != nil {
		return err
	}
	if err := RegisterMirrorPlugin(); err != nil {
		return err
	}
	if err := RegisterDlt645Plugin(); err != nil {
		return err
	}
	if err := RegisterGatewayPlugin(); err != nil {
		return err
	}
	return nil
}
