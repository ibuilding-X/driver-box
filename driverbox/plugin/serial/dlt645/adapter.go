package dlt645

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/driverbox/plugin/callback"
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

// 采集点位组
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

	//生成点位采集组
	for _, model := range connector.Plugin.Config.DeviceModels {
		for _, dev := range model.Devices {
			if dev.ConnectionKey != adapter.connector.ConnectionKey {
				continue
			}
			adapter.createPointGroup(model, dev)
		}
	}

	groups := make([]serial.TimerGroup, 0)
	for _, group := range adapter.groups {
		groups = append(groups, group.TimerGroup)
		devices := make(map[string]string)
		for _, point := range group.Points {
			if _, ok := devices[point.DeviceId]; !ok {
				group.Devices = append(group.Devices, point.DeviceId)
				devices[point.DeviceId] = point.DeviceId
			}
		}
	}
	return groups
}

// 采集任务分组
func (adapter *dlt645Adapter) createPointGroup(model config.DeviceModel, dev config.Device) {
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
		unitID := dev.Properties["slaveId"]
		if len(unitID) == 0 {
			helper.Logger.Error("unitID not found", zap.String("deviceId", dev.ID))
			continue
		}
		key := unitID + "/" + ext.DataMaker
		group := adapter.groups[key]

		if group == nil {
			group = &pointGroup{
				TimerGroup: serial.TimerGroup{
					UUID:     key,
					Duration: duration,
					Devices:  make([]string, 0),
				},
				UnitID:    unitID,
				DataMaker: ext.DataMaker,
				Points:    make([]*Point, 0),
			}
			adapter.groups[key] = group
		} else if group.TimerGroup.Duration > duration {
			group.TimerGroup.Duration = duration
		}
		group.Points = append(group.Points, ext)
	}
}

func (adapter *dlt645Adapter) ExecuteTimerGroup(group *serial.TimerGroup) error {
	helper.Logger.Info("timer group", zap.Any("group", adapter.groups[group.UUID]))
	g := adapter.groups[group.UUID]
	cmd := serial.Command{
		Mode:        plugin.ReadMode,
		UUId:        group.UUID,
		OutputFrame: getOutputFrame(g.UnitID, g.Points[0].DataMaker),
		Callback: func(inputFrame []byte) error {
			backData := fmt.Sprintf("[% x]", inputFrame)
			value, _ := analysis(backData)
			helper.Logger.Info("decode", zap.Any("frame", backData), zap.Any("data", value))
			deviceData := make([]plugin.DeviceData, 0)
			for _, point := range g.Points {
				deviceData = append(deviceData, plugin.DeviceData{
					ID:     point.DeviceId,
					Values: []plugin.PointData{{PointName: point.Name, Value: value}},
				})
			}
			callback.ExportTo(deviceData)
			return nil
		},
	}
	helper.Logger.Info("outputFrame", zap.Any("outputFrame", cmd.OutputFrame))
	return adapter.connector.Send(cmd)
}
func (adapter *dlt645Adapter) DriverBoxEncode(deviceId string, mode plugin.EncodeMode, values ...plugin.PointData) (res []serial.Command, err error) {
	return nil, errors.New("not support")
}

func (adapter *dlt645Adapter) SendCommand(cmd serial.Command) error {
	request := strings.Replace(cmd.OutputFrame, " ", "", -1)
	serialMessage, _ := hex.DecodeString(request)
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
	return cmd.Callback(data[0:sum])
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
	meterNumberHandle, _ := hex.DecodeString(MeterNumber)
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
	DataMarkerHandle, _ := hex.DecodeString(DataMarker)
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

	messageFinished := "68 " + meterNumberHandleReverseFinished + " 68" + " 11 " + "04 " + DataMarkerHandleReverseFinished
	return CheckCode(messageFinished)
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
	//如果校验位长度不够，就补0，因为校验位必须是要2位
	if len(hex) < 2 {
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
