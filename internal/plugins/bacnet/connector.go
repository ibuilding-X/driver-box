package bacnet

import (
	"encoding/json"
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/common"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/helper/crontab"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/internal/plugins/bacnet/bacnet"
	"github.com/ibuilding-x/driver-box/internal/plugins/bacnet/bacnet/btypes"
	"github.com/ibuilding-x/driver-box/internal/plugins/bacnet/bacnet/network"
	"github.com/spf13/cast"
	"go.uber.org/zap"
	"net"
	"time"
)

const (
	IP = "ip"
)

type connector struct {
	key     string
	plugin  *Plugin
	network *network.Network
	//通讯设备集合
	devices map[string]*device
	//当前连接的定时扫描任务
	collectTask *crontab.Future
	close       bool
}

// 采集组
type device struct {
	// 通讯设备，采集点位可以对应多个物模型设备
	device *network.Device
	//分组
	pointGroup []*pointGroup
}

type pointGroup struct {
	//采集间隔
	Duration time.Duration
	//上一次采集时间
	LatestTime time.Time
	multiData  *btypes.MultiplePropertyData
}

// initCollectTask 启动数据采集任务
func (c *connector) initCollectTask(bic *bacIpConfig) (err error) {
	for _, model := range c.plugin.config.DeviceModels {
		for _, dev := range model.Devices {
			if dev.ConnectionKey != c.key {
				continue
			}
			for _, point := range model.DevicePoints {
				p := point.ToPoint()
				if p.ReadWrite != config.ReadWrite_R && p.ReadWrite != config.ReadWrite_RW {
					continue
				}
				var ext extends
				if err = helper.Map2Struct(p.Extends, &ext); err != nil {
					helper.Logger.Error("error bacnet config", zap.Any("config", p.Extends), zap.Error(err))
					continue
				}
				//未设置，则默认每秒采集一次
				if ext.Duration == "" {
					ext.Duration = "1s"
				}
				duration, err := time.ParseDuration(ext.Duration)
				if err != nil {
					helper.Logger.Error("error bacnet duration config", zap.String("deviceSn", dev.DeviceSn), zap.Any("config", p.Extends), zap.Error(err))
					duration = time.Second
				}

				object, err := createObject(ext)
				if err != nil {
					helper.Logger.Error("error bacnet config", zap.Any("config", p.Extends), zap.Error(err))
					continue

				}
				object.Name = p.Name           //通信时未携带该参数
				object.DeviceSn = dev.DeviceSn //通信时未携带该参数

				device, err := c.createDevice(dev.Properties)
				ok := false
				for _, group := range device.pointGroup {
					//相同采集频率为同一组
					if group.Duration != duration {
						continue
					}
					//暂定每批最多20个点
					if len(group.multiData.Objects) >= 15 {
						continue
					}
					ok = true
					group.multiData.Objects = append(group.multiData.Objects, object)
				}
				//新增一个点位组
				if !ok {
					device.pointGroup = append(device.pointGroup, &pointGroup{
						Duration: duration,
						multiData: &btypes.MultiplePropertyData{
							Objects: []btypes.Object{
								object,
							},
						},
					})
				}
			}
		}
	}

	duration := bic.Duration
	if duration == "" {
		helper.Logger.Warn("bacnet connection duration is empty, use default 5s", zap.String("key", c.key))
		duration = "5s"
	}

	future, err := helper.Crontab.AddFunc(duration, func() {
		//遍历所有通讯设备
		for deviceId, device := range c.devices {
			if len(device.pointGroup) == 0 {
				helper.Logger.Warn("device has none read point", zap.String("device", deviceId))
				continue
			}
			//批量遍历通讯设备下的点位，并将结果关联至物模型设备
			for i, group := range device.pointGroup {
				if c.close {
					helper.Logger.Warn("bacnet connection is closed, ignore collect task!", zap.String("key", c.key))
					break
				}

				if group.LatestTime.Add(group.Duration).After(time.Now()) {
					continue
				}
				//采集时间未到
				helper.Logger.Debug("timer read bacnet", zap.Any("group", i), zap.Any("latestTime", group.LatestTime), zap.Any("duration", group.Duration))
				bac := bacRequest{
					deviceId: deviceId,
					mode:     plugin.ReadMode,
					req: btypes.MultiplePropertyData{
						Objects: group.multiData.Objects,
					},
				}
				err := c.Send(bac)
				group.LatestTime = time.Now()
				if err != nil {
					helper.Logger.Error("read error", zap.Error(err))
				}
			}

		}
	})
	if err != nil {
		return err
	} else {
		c.collectTask = future
		return nil
	}
}

func (c *connector) Send(raw interface{}) (err error) {
	br := raw.(bacRequest)
	device, ok := c.devices[br.deviceId]
	if !ok {
		helper.Logger.Error("none device config")
		return err
	}
	switch br.mode {
	// 读
	case plugin.ReadMode:
		req := br.req.(btypes.MultiplePropertyData)
		out, err := device.device.ReadMuti(req)
		if err != nil {
			return err
		}
		if out.ErrorClass != 0 {
			c.plugin.logger.Error(fmt.Sprintf("read error: [%d-%d] %s)", out.ErrorClass, out.ErrorCode, err.Error()))
			return err
		}
		for _, object := range out.Objects {
			resp, err := convertObj2Resp(&object)
			if err != nil {
				helper.Logger.Error("error bacnet result", zap.Any("object", object), zap.Error(err))
				continue
			}

			for _, obj := range req.Objects {
				if obj.ID == object.ID {
					resp.PointName = obj.Name
					resp.DeviceSn = obj.DeviceSn
					break
				}
			}

			respJson, err := json.Marshal(resp)
			_, err = c.plugin.callback(c.plugin, string(respJson))
			if err != nil {
				helper.Logger.Error("error bacnet callback", zap.Any("data", respJson), zap.Error(err))
			}
		}
	case plugin.WriteMode:
		write := br.req.(*network.Write)
		if err := device.device.Write(write); err != nil {
			c.plugin.logger.Error(fmt.Sprintf("write error: %s", err.Error()))
			return err
		}

		//触发读操作，及时或许最新值
		point, ok := helper.CoreCache.GetPointByDevice(write.DeviceSn, write.PointName)
		if ok && (point.ReadWrite == config.ReadWrite_R || point.ReadWrite == config.ReadWrite_RW) {
			//延迟100毫秒触发读操作
			go func(write *network.Write) {
				i := 0
				for i < 10 {
					i++
					time.Sleep(time.Duration(i*100) * time.Millisecond)
					helper.Logger.Info("point write success,try to read new value", zap.String("point", write.PointName))
					err = helper.Send(write.DeviceSn, plugin.ReadMode, plugin.PointData{
						PointName: write.PointName,
					})
					if err != nil {
						helper.Logger.Error("point write success, read new value error", zap.String("point", write.PointName), zap.Error(err))
						break
					}

					value, _ := helper.DeviceShadow.GetDevicePoint(write.DeviceSn, write.PointName)
					helper.Logger.Info("point write success, read new value", zap.String("point", write.PointName), zap.Any("expect", write.WriteValue), zap.Any("value", value))
					if fmt.Sprint(write.WriteValue) == fmt.Sprint(value) {
						break
					}
				}
			}(write)
		}
	default:
		return common.NotSupportMode
	}
	return nil
}

type readResponse struct {
	Value     interface{}       `json:"value"`
	Status    map[string]string `json:"status"`
	DeviceSn  string            `json:"deviceSn"`
	PointName string            `json:"pointName"`
}

func convertObj2Resp(object *btypes.Object) (resp *readResponse, err error) {
	resp = &readResponse{}
	for _, prop := range object.Properties {
		switch prop.Type {
		case btypes.PROP_PRESENT_VALUE:
			resp.Value = prop.Data
		case btypes.PROP_STATUS_FLAGS:
			status := make(map[string]string)
			bitValues := prop.Data.(*btypes.BitString).GetValue()
			status["alarm"] = cast.ToString(bitValues[0])
			status["fault"] = cast.ToString(bitValues[1])
			status["overridden"] = cast.ToString(bitValues[2])
			status["outofservice"] = cast.ToString(bitValues[3])
			resp.Status = status
		}
	}
	if resp.Value == nil {
		return nil, fmt.Errorf("read value is nil")
	}
	return resp, nil
}

func (c *connector) Release() (err error) {
	return nil
}

func (c *connector) Close() {
	c.close = true
	c.collectTask.Disable()
	c.network.NetworkClose()
}

// deviceProtocol 设备协议部分
type deviceProtocol struct {
	Ip            string `json:"ip"`
	Port          string `json:"port"`
	Id            string `json:"id"`
	NetworkNumber string `json:"networkNumber"`
	MacMstp       string `json:"macMstp"`
	MaxApdu       string `json:"maxApdu"`
	Segmentation  string `json:"segmentation"`
}

// createDevice
func (c *connector) createDevice(properties map[string]string) (d *device, err error) {
	var dp deviceProtocol
	if err = helper.Map2Struct(properties, &dp); err != nil {
		return nil, err
	}
	if dp.Port == "" {
		dp.Port = "47808"
	}
	if dp.NetworkNumber == "" {
		dp.NetworkNumber = "0"
	}
	if dp.MaxApdu == "" {
		dp.MaxApdu = "1476"
	}
	if dp.MacMstp == "" {
		dp.MacMstp = "0"
	}
	if dp.Segmentation == "" {
		dp.Segmentation = "0"
	}
	//复用缓存
	d, ok := c.devices[dp.Id]
	if ok {
		return d, nil
	}
	//新增设备连接
	dev, err := network.NewDevice(c.network, &network.Device{
		Ip:            dp.Ip,
		Port:          cast.ToInt(dp.Port),
		DeviceID:      cast.ToInt(dp.Id),
		NetworkNumber: cast.ToInt(dp.NetworkNumber),
		MacMSTP:       cast.ToInt(dp.MacMstp),
		MaxApdu:       cast.ToUint32(dp.MaxApdu),
		Segmentation:  cast.ToUint32(dp.Segmentation),
	})
	if err != nil {
		return nil, err
	}
	var group []*pointGroup
	d = &device{
		device:     dev,
		pointGroup: group,
	}
	c.devices[dp.Id] = d
	return d, nil
}

// bacnet connection配置项
type bacIpConfig struct {
	Interface   string `json:"interface"`
	LocalIp     string `json:"localIp"`
	LocalSubnet int    `json:"localSubnet"`
	LocalPort   int    `json:"localPort"`
	Duration    string `json:"duration"` // 自动采集周期
}

func initConnector(key string, config map[string]interface{}, p *Plugin) (*connector, error) {
	// 获取网卡信息
	if mode, ok := config["mode"]; ok {
		switch mode {
		case IP:
			var bic bacIpConfig
			if err := helper.Map2Struct(config, &bic); err == nil {
				var n *network.Network
				if err = bic.checkBacIpConfig(); err != nil {
					return nil, err
				}
				if bic.Interface != "" {
					if n, err = network.New(&network.Network{
						Interface: bic.Interface,
						Port:      bic.LocalPort,
					}); err != nil {
						return nil, err
					}
				} else {
					if n, err = network.New(&network.Network{
						Ip:         bic.LocalIp,
						SubnetCIDR: bic.LocalSubnet,
						Port:       bic.LocalPort,
					}); err != nil {
						return nil, err
					}
				}
				n.NetworkRun()

				c := &connector{
					key:     key,
					network: n,
					plugin:  p,
					devices: make(map[string]*device),
				}
				//启动数据采集任务
				err = c.initCollectTask(&bic)
				if err != nil {
					helper.Logger.Error("init Collect Task error", zap.Error(err))
				}
				return c, err
			} else {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("%s not supported currently", mode)
		}
	} else {
		return nil, fmt.Errorf("mode is required")
	}
}

func (bic *bacIpConfig) checkBacIpConfig() error {
	if bic.LocalPort == 0 {
		if freePort, err := getFreePort(); err == nil {
			bic.LocalPort = freePort
		} else {
			return err
		}
	}
	if bic.Interface == "" && bic.LocalIp == "" {
		return fmt.Errorf("bic ip config error: %+v", bic)
	}
	return nil
}

// getFreePort 获取未被使用的端口
func getFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", ":0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func createObject(ext extends) (btypes.Object, error) {
	if !validObjType(ext.ObjType) {
		return btypes.Object{}, fmt.Errorf("unsupported objType: %s", ext.ObjType)
	}
	return btypes.Object{
		ID: btypes.ObjectID{
			Type:     btypes.GetType(ext.ObjType),
			Instance: btypes.ObjectInstance(ext.Ins),
		},
		Properties: []btypes.Property{
			{
				Type:       btypes.PropPresentValue,
				ArrayIndex: bacnet.ArrayAll,
			},
			{
				Type:       btypes.PROP_STATUS_FLAGS,
				ArrayIndex: bacnet.ArrayAll,
			},
		},
	}, nil
}
