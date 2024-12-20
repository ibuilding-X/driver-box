package dlt645

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/driverbox/plugin/serial"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"io"
	"math/big"
	"strconv"
	"strings"
	"time"
)

var Adapter = &dlt645Adapter{}

const (
	pduMinSize = 1   // funcCode(1)
	pduMaxSize = 253 // funcCode(1) + data(252)

	rtuAduMinSize = 4   // address(1) + funcCode(1) + crc(2)
	rtuAduMaxSize = 256 // address(1) + PDU(253) + crc(2)

	asciiAduMinSize       = 3
	asciiAduMaxSize       = 256
	asciiCharacterMaxSize = 513

	tcpProtocolIdentifier = 0x0000

	tcpHeaderMbapSize = 7 // MBAP header
	tcpAduMinSize     = 8 // MBAP + funcCode
	tcpAduMaxSize     = 260
)

// 采集组
type slaveDevice struct {
	// 通讯设备，采集点位可以对应多个物模型设备
	unitID string
	//分组
	pointGroup []*pointGroup
}

type pointGroup struct {
	serial.TimerGroup
	// 从机地址
	UnitID    string
	DataMaker string // dlt645标准中点位标识
	Points    []*Point
}

type Point struct {
	config.Point
	//冗余设备相关信息
	DeviceId  string
	DataMaker string `json:"dataMaker"` // dlt645标准中点位标识
	//点位采集周期
	Duration string `json:"duration"`
}

type dlt645Adapter struct {
	connector *serial.Connector
	groups    map[string]*pointGroup
}

func (adapter *dlt645Adapter) InitTimerGroup(connector *serial.Connector) []serial.TimerGroup {
	helper.Logger.Info("init modbus adapter")
	adapter.connector = connector
	adapter.groups = make(map[string]*pointGroup)

	devices := make(map[string]*slaveDevice)

	//生成点位采集组
	for _, model := range connector.Plugin.Config.DeviceModels {
		for _, dev := range model.Devices {
			if dev.ConnectionKey != adapter.connector.ConnectionKey {
				continue
			}
			adapter.createPointGroup(model, dev, devices)
		}
	}

	groups := make([]serial.TimerGroup, 0)
	for _, device := range devices {
		for _, group := range device.pointGroup {
			groups = append(groups, group.TimerGroup)
			adapter.groups[group.UUID] = group
		}
	}
	return groups
}

// 采集任务分组
func (adapter *dlt645Adapter) createPointGroup(model config.DeviceModel, dev config.Device, devices map[string]*slaveDevice) {
	for _, point := range model.DevicePoints {
		p := point.ToPoint()
		if p.ReadWrite != config.ReadWrite_R && p.ReadWrite != config.ReadWrite_RW {
			continue
		}
		ext := new(Point)
		if err := helper.Map2Struct(p.Extends, ext); err != nil {
			helper.Logger.Error("error serial config", zap.Any("config", ext), zap.Error(err))
			continue
		}
		//未设置，则默认每秒采集一次
		if ext.Duration == "" {
			ext.Duration = "1s"
		}
		ext.Name = p.Name
		ext.DeviceId = dev.ID
		duration, err := time.ParseDuration(ext.Duration)
		if err != nil {
			helper.Logger.Error("error modbus duration config", zap.String("deviceId", dev.ID), zap.Any("config", p.Extends), zap.Error(err))
			duration = time.Second
		}

		device, err := adapter.createDevice(dev.Properties, devices)
		if err != nil {
			helper.Logger.Error("error modbus device config", zap.String("deviceId", dev.ID), zap.Any("config", p.Extends), zap.Error(err))
			continue
		}
		ok := false
		for _, group := range device.pointGroup {
			//相同采集频率为同一组
			if group.Duration != duration {
				continue
			}
			if group.DataMaker == ext.DataMaker {
				ok = true
				break
			}
		}
		//新增一个点位组
		if !ok {
			ext.DeviceId = dev.ID
			ext.Name = p.Name
			device.pointGroup = append(device.pointGroup, &pointGroup{
				TimerGroup: serial.TimerGroup{
					UUID:     uuid.New().String(),
					Duration: duration,
				},
				UnitID: device.unitID,
				Points: []*Point{
					ext,
				},
			})
		}
	}
}
func (adapter *dlt645Adapter) createDevice(properties map[string]string, devices map[string]*slaveDevice) (d *slaveDevice, err error) {
	unitID, ok := properties["slaveId"]
	if !ok {
		return nil, errors.New("unitID not found")
	}
	d, ok = devices[unitID]
	if ok {
		return d, nil
	}

	var group []*pointGroup
	d = &slaveDevice{
		unitID:     unitID,
		pointGroup: group,
	}
	devices[unitID] = d
	return d, nil
}

func (adapter *dlt645Adapter) ExecuteTimerGroup(group *serial.TimerGroup) error {
	helper.Logger.Info("timer group", zap.Any("group", adapter.groups[group.UUID]))
	g := adapter.groups[group.UUID]
	cmd := serial.Command{
		Mode:        plugin.ReadMode,
		OutputFrame: getOutputFrame(g.UnitID, g.Points[0].DataMaker),
	}
	helper.Logger.Info("outputFrame", zap.Any("outputFrame", cmd.OutputFrame))
	time.Sleep(time.Second)
	return adapter.connector.Send(cmd)
}
func (adapter *dlt645Adapter) DriverBoxEncode(deviceId string, mode plugin.EncodeMode, values ...plugin.PointData) (res []serial.Command, err error) {
	return nil, nil
}
func (adapter *dlt645Adapter) DriverBoxDecode(command serial.Command) (res []plugin.DeviceData, err error) {
	return nil, nil
}
func (adapter *dlt645Adapter) SendCommand(cmd serial.Command) error {
	request := strings.Replace(cmd.OutputFrame, " ", "", -1)
	serialMessage := HexStringToBytes(request)
	helper.Logger.Info("sending ", zap.String("frame", fmt.Sprintf("[% x]", serialMessage)))
	_, err := adapter.connector.Client.Write(serialMessage)
	if err != nil {
		return err
	}
	var data [rtuAduMaxSize]byte

	sum, err := io.ReadFull(&adapter.connector.Client, data[:])
	if sum < 0 {
		return nil
	}
	backData := fmt.Sprintf("[% x]", data[0:sum])
	value, err := analysis(backData)
	helper.Logger.Info("decode", zap.Any("frame", backData), zap.Any("data", value))
	return nil
}
func analysis(command string) (float64, error) {
	command = strings.Replace(command, "[", "", -1)
	command = strings.Replace(command, "]", "", -1)
	newCommands := strings.Split(command, " ")

	//跳过FE字段
	i := 0
	for ; i < len(newCommands) && (newCommands[i] == "FE" || newCommands[i] == "fe"); i++ {
		i++
	}
	newCommands = newCommands[i:]

	start, _ := strconv.Atoi(newCommands[0])
	end, _ := strconv.Atoi(newCommands[len(newCommands)-1])
	if len(newCommands) < 16 || len(newCommands) > 26 || start != 68 || end != 16 {
		return 0, fmt.Errorf("invalid response")
	} else {
		helper.Logger.Debug(fmt.Sprintf("报文源码：%s", command))
		helper.Logger.Debug(fmt.Sprintf("帧起始符：%s", newCommands[0]))
		helper.Logger.Debug(fmt.Sprintf("报文源码：%s", command))
		helper.Logger.Debug(fmt.Sprintf("报文源码：%s", command))
		helper.Logger.Debug(fmt.Sprintf("报文源码：%s", command))
		//meter_serial := newCommands[6] + newCommands[5] + newCommands[4] + newCommands[3] + newCommands[2] + newCommands[1]

		//逆序传输的，且需要统一逐个减去十六进制0x33后才是真实值
		hexSub33 := func(hexStr string) string {
			value := new(big.Int)
			value.SetString(hexStr, 16)
			value.Sub(value, big.NewInt(0x33))
			return fmt.Sprintf("%02x", value)
		}

		dltDataFinished := hexSub33(newCommands[13])
		dltDataFinished1 := hexSub33(newCommands[12])
		//dltDataFinished2 := hexSub33(newCommands[11])
		dltDataFinished3 := hexSub33(newCommands[10])

		//makers := dltDataFinished + dltDataFinished1 + dltDataFinished2 + dltDataFinished3

		dataUnits := len(newCommands) - 2
		var data string
		for i := dataUnits; i > 14; i-- {
			midData := hexSub33(newCommands[i-1])
			data += fmt.Sprintf("%s", midData)
		}

		// 原始值
		n1, _ := decimal.NewFromString(data)
		// 系数
		var n2 decimal.Decimal

		switch {
		case dltDataFinished == "00":
			n2, _ = decimal.NewFromString("0.01")
		case dltDataFinished == "02":
			switch dltDataFinished1 {
			case "01":
				n2, _ = decimal.NewFromString("0.1")
			case "02", "06":
				n2, _ = decimal.NewFromString("0.001")
			case "03", "04", "05":
				n2, _ = decimal.NewFromString("0.0001")
			}
			if dltDataFinished3 == "02" {
				n2, _ = decimal.NewFromString("0.01")
			}
		}

		over := n1.Mul(n2)
		val, _ := over.Float64()
		return val, nil
	}
}

func (adapter *dlt645Adapter) Release() (err error) {
	return err
}

func getOutputFrame(MeterNumber string, DataMarker string) string {
	//表号
	meterNumberHandle := HexStringToBytes(MeterNumber)
	meterNumberHandleX := fmt.Sprintf("% x", meterNumberHandle)
	meterNumberHandleReverse := strings.Split(meterNumberHandleX, " ")
	for i := 0; i < len(meterNumberHandleReverse)/2; i++ {
		mid := meterNumberHandleReverse[i]
		meterNumberHandleReverse[i] = meterNumberHandleReverse[len(meterNumberHandleReverse)-1-i]
		meterNumberHandleReverse[len(meterNumberHandleReverse)-1-i] = mid
	}
	midMeterNumberHandle := fmt.Sprintf("% s", meterNumberHandleReverse)
	meterNumberHandleReverseFinished := strings.Replace(midMeterNumberHandle, "[", "", -1)
	meterNumberHandleReverseFinished = strings.Replace(meterNumberHandleReverseFinished, "]", "", -1)
	//数据标识
	DataMarkerHandle := HexStringToBytes(DataMarker)
	DataMarkerHandleX := fmt.Sprintf("% x", DataMarkerHandle)
	DataMarkerHandleReverse := strings.Split(DataMarkerHandleX, " ")

	//反转后的数据标识
	result := make([]string, len(DataMarkerHandleReverse))

	for i := 0; i < len(DataMarkerHandleReverse)/2; i++ {
		process := func(hexStr string) string {
			value := new(big.Int)
			value.SetString(hexStr, 16)
			value.Add(value, big.NewInt(0x33))
			return fmt.Sprintf("%02x", value)
		}
		result[i] = process(DataMarkerHandleReverse[len(DataMarkerHandleReverse)-i-1])
		result[len(DataMarkerHandleReverse)-i-1] = process(DataMarkerHandleReverse[i])
	}
	midDataMarkerHandle := fmt.Sprintf("% s", result)
	DataMarkerHandleReverseFinished := strings.Replace(midDataMarkerHandle, "[", "", -1)
	DataMarkerHandleReverseFinished = strings.Replace(DataMarkerHandleReverseFinished, "]", "", -1)

	messageFinshed := "68 " + meterNumberHandleReverseFinished + " 68" + " 11 " + "04 " + DataMarkerHandleReverseFinished
	return CheckCode(messageFinshed)
}

// 计算出校验码
func CheckCode(data string) string {
	midData := data
	data = strings.ReplaceAll(data, " ", "")
	total := 0
	length := len(data)
	num := 0
	for num < length {
		s := data[num : num+2]
		//16进制转换成10进制
		totalMid, _ := strconv.ParseUint(s, 16, 32)
		total += int(totalMid)
		num = num + 2
	}
	//将校验码前面的所有数通过16进制加起来转换成10进制，然后除256区余数，然后余数转换成16进制，得到的就是校验码
	mod := total % 256
	hex, _ := DecConvertToX(mod, 16)
	len := len(hex)
	//如果校验位长度不够，就补0，因为校验位必须是要2位
	if len < 2 {
		hex = "0" + hex
	}
	return midData + " " + strings.ToUpper(hex) + " 16"
}
func DecConvertToX(n, num int) (string, error) {
	if n < 0 {
		return strconv.Itoa(n), errors.New("只支持正整数")
	}
	if num != 2 && num != 8 && num != 16 {
		return strconv.Itoa(n), errors.New("只支持二、八、十六进制的转换")
	}
	result := ""
	h := map[int]string{
		0:  "0",
		1:  "1",
		2:  "2",
		3:  "3",
		4:  "4",
		5:  "5",
		6:  "6",
		7:  "7",
		8:  "8",
		9:  "9",
		10: "A",
		11: "B",
		12: "C",
		13: "D",
		14: "E",
		15: "F",
	}
	for ; n > 0; n /= num {
		lsb := h[n%num]
		result = lsb + result
	}
	return result, nil
}
func HexStringToBytes(data string) []byte {
	if "" == data {
		return nil
	}
	data = strings.ToUpper(data)
	length := len(data) / 2
	dataChars := []byte(data)
	var byteData []byte = make([]byte, length)
	for i := 0; i < length; i++ {
		pos := i * 2
		byteData[i] = byte(charToByte(dataChars[pos])<<4 | charToByte(dataChars[pos+1]))
	}
	return byteData

}
func charToByte(c byte) byte {
	return (byte)(strings.Index("0123456789ABCDEF", string(c)))
}
