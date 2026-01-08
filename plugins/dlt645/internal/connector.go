package internal

import (
	"errors"
	"time"

	"github.com/ibuilding-x/driver-box/driverbox"
	"github.com/ibuilding-x/driver-box/driverbox/common"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/helper/crontab"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	dlt "github.com/ibuilding-x/driver-box/plugins/dlt645/internal/core"
	"github.com/ibuilding-x/driver-box/plugins/dlt645/internal/core/dltcon"
	"go.uber.org/zap"
)

func newConnector(p *Plugin, cf *ConnectionConfig) (*connector, error) {
	if cf.MinInterval == 0 {
		cf.MinInterval = 100
	}
	if cf.Retry == 0 {
		cf.Retry = 3
	}
	if cf.Timeout <= 0 {
		cf.Timeout = 1000
	}

	provider := dlt.NewClientProvider()
	provider.Address = cf.Address
	provider.BaudRate = int(cf.BaudRate)
	provider.DataBits = int(cf.DataBits)
	provider.Parity = cf.Parity
	provider.StopBits = int(cf.StopBits)
	provider.Timeout = time.Duration(cf.Timeout) * time.Millisecond

	client := dltcon.NewClient(provider)
	client.LogMode(cf.ProtocolLogEnabled)
	if cf.AutoReconnect {
		client.SetAutoReconnect(1)
	} else {
		client.SetAutoReconnect(0)
	}
	//cf.Ls = p.ls
	//cf.ScriptEnable = helper.ScriptExists(p.config.Key)
	conn := &connector{
		config:  cf,
		plugin:  p,
		client:  client,
		virtual: cf.Virtual,
		devices: make(map[string]*slaveDevice),
	}

	err := client.Start()

	return conn, err
}

func (c *connector) initCollectTask(conf *ConnectionConfig) (*crontab.Future, error) {
	if !conf.Enable {
		helper.Logger.Warn("dlt645 connection is disabled, ignore collect task", zap.String("key", c.config.ConnectionKey))
		return nil, nil
	}
	if len(c.devices) == 0 {
		helper.Logger.Warn("dlt645 connection has no device to collect", zap.String("key", c.config.ConnectionKey))
		return nil, nil
	}

	//注册定时采集任务
	return helper.Crontab.AddFunc("1s", func() {
		//遍历所有通讯设备
		for unitID, device := range c.devices {
			if len(device.pointGroup) == 0 {
				helper.Logger.Warn("device has none read point", zap.String("slaveId", unitID))
				continue
			}
			//批量遍历通讯设备下的点位，并将结果关联至物模型设备
			for i, group := range device.pointGroup {
				if c.close {
					helper.Logger.Warn("dlt645 connection is closed, ignore collect task!", zap.String("key", c.config.ConnectionKey))
					break
				}

				//采集时间未到
				if group.LatestTime.Add(group.Duration).After(time.Now()) {
					continue
				}

				helper.Logger.Debug("timer read dlt645", zap.Any("group", i), zap.Any("latestTime", group.LatestTime), zap.Any("duration", group.Duration))
				bac := command{
					Mode:  plugin.ReadMode,
					Value: group,
				}
				if err := c.Send(bac); err != nil {
					helper.Logger.Error("read error", zap.Any("connection", conf), zap.Any("group", group), zap.Error(err))
					//通讯失败，触发离线
					devices := make(map[string]interface{})
					for _, point := range group.Points {
						if devices[point.DeviceId] != nil {
							continue
						}
						devices[point.DeviceId] = point.Name
						_ = helper.DeviceShadow.MayBeOffline(point.DeviceId)
					}
				}
				group.LatestTime = time.Now()
			}

		}
	})
}

// 采集任务分组
func (c *connector) createPointGroup(conf *ConnectionConfig, model config.DeviceModel, dev config.Device) {
	groupIndex := 0
	for _, point := range model.DevicePoints {
		if point.ReadWrite() != config.ReadWrite_R && point.ReadWrite() != config.ReadWrite_RW {
			continue
		}
		ext, err := convToPointExtend(point)
		if err != nil {
			helper.Logger.Error("error dlt645 point config", zap.String("deviceId", dev.ID), zap.Any("point", point), zap.Error(err))
			continue
		}
		ext.DeviceId = dev.ID
		duration, err := time.ParseDuration(ext.Duration)
		if err != nil {
			helper.Logger.Error("error dlt645 duration config", zap.String("deviceId", dev.ID), zap.Any("config", point), zap.Error(err))
			duration = time.Second
		}

		device, err := c.createDevice(dev.Properties)
		if err != nil {
			helper.Logger.Error("error dlt645 device config", zap.String("deviceId", dev.ID), zap.Any("config", point), zap.Error(err))
			continue
		}

		ext.DeviceId = dev.ID
		device.pointGroup = append(device.pointGroup, &pointGroup{
			index:    groupIndex,
			Duration: duration,
			Quantity: ext.Quantity,
			Points: []*Point{
				ext,
			},
			SlaveId:   dev.Properties["slaveId"],
			DataMaker: ext.DataMaker,
		})
		groupIndex++
	}

}

// Send 发送数据
func (c *connector) Send(data interface{}) (err error) {
	cmd := data.(command)
	switch cmd.Mode {
	// 读
	case plugin.ReadMode:
		group := cmd.Value.(*pointGroup)
		return c.sendReadCommand(group)
	case plugin.WriteMode:
	default:
		return common.NotSupportMode
	}

	return
}

// Release 释放资源
func (c *connector) Release() (err error) {
	return
}

func (c *connector) Close() {
	c.close = true
	if c.collectTask != nil {
		c.collectTask.Disable()
	}
	_ = c.client.Close()
}

// ensureInterval 确保与前一次IO至少间隔minInterval毫秒
func (c *connector) ensureInterval() {
	np := c.latestIoTime.Add(time.Duration(c.config.MinInterval) * time.Millisecond)
	if time.Now().Before(np) {
		time.Sleep(time.Until(np))
	}
	c.latestIoTime = time.Now()
}

func (c *connector) sendReadCommand(group *pointGroup) error {

	var value float64
	var err error
	if c.virtual {
		value = 0
	} else {
		value, err = c.read(group.SlaveId, group.DataMaker)
	}

	if err != nil {
		return err
	}
	// 转化数据并上报
	for _, point := range group.Points {
		pointReadValue := plugin.PointReadValue{
			ID:        point.DeviceId,
			PointName: point.Name(),
			Value:     value,
		}
		res, err := c.Decode(pointReadValue)
		if err != nil {
			helper.Logger.Error("error dlt645 callback", zap.Any("data", pointReadValue), zap.Error(err))
		} else {
			driverbox.ExportTo(res)
		}
	}
	return nil
}

func (c *connector) resetCollectTime(group *pointGroup) {
	for _, device := range c.devices {
		if device.address == group.Address {
			for _, g := range device.pointGroup {
				g.LatestTime = time.Now().Add(-group.Duration)
			}
			break
		}
	}
}

func (c *connector) sendWriteCommand() error {
	return nil
}

// read 读操作
// 首次读取失败，将尝试重连 dlt645 连接
func (c *connector) read(address, dataMaker string) (values float64, err error) {
	readconfig := &dlt.Dlt645ConfigClient{address, dataMaker}
	value, err := readconfig.SendMessageToSerial(c.client)
	if err != nil {
		helper.Logger.Error("dlt645 client error", zap.Any("dlt645", c.config), zap.Error(err))
	}
	return value, err
}

// write 写操作
func (c *connector) write() (err error) {
	return
}

func convToPointExtend(extends config.Point) (*Point, error) {
	extend := new(Point)
	extend.Point = extends
	if err := helper.Map2Struct(extends, extend); err != nil {
		helper.Logger.Error("error dlt645 config", zap.Any("config", extends), zap.Error(err))
		return nil, err
	}
	//未设置，则默认每秒采集一次
	if extend.Duration == "" {
		extend.Duration = "1s"
	}
	return extend, nil
}

func (c *connector) createDevice(properties map[string]string) (d *slaveDevice, err error) {
	address, err := getMeterAddress(properties)
	d, ok := c.devices[address]
	if ok {
		return d, nil
	}

	var group []*pointGroup
	d = &slaveDevice{
		address:    address,
		pointGroup: group,
	}
	c.devices[address] = d
	return d, nil
}

func getMeterAddress(properties map[string]string) (string, error) {
	address := properties["slaveId"]
	if len(address) == 0 {
		return "", errors.New("none address")
	}
	return address, nil
}
