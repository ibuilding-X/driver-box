package dlt645

import (
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/ibuilding-x/driver-box/v2/driverbox"
	"github.com/shopspring/decimal"
)

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

func analysis(dlt *Dlt645ClientProvider, command string) (float64, error) {
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
		dlt.Debug("invalid response", newCommands)
		return 0, fmt.Errorf("invalid response")
	} else {
		driverbox.Log().Debug(fmt.Sprintf("报文源码：%s", command))
		driverbox.Log().Debug(fmt.Sprintf("帧起始符：%s", newCommands[0]))
		driverbox.Log().Debug(fmt.Sprintf("报文源码：%s", command))
		driverbox.Log().Debug(fmt.Sprintf("报文源码：%s", command))
		driverbox.Log().Debug(fmt.Sprintf("报文源码：%s", command))
		dlt.Debug("帧起始符：%s", newCommands[0])
		meter_serial := newCommands[6] + newCommands[5] + newCommands[4] + newCommands[3] + newCommands[2] + newCommands[1]
		dlt.Debug("电表地址：%s", meter_serial)
		dlt.Debug("控制域：%s", newCommands[8])
		dlt.Debug("数据域长度：%s", newCommands[9])
		dlt.Debug("校验码：%s", newCommands[len(newCommands)-2])
		dlt.Debug("停止位：%s", newCommands[len(newCommands)-1])

		//逆序传输的，且需要统一逐个减去十六进制0x33后才是真实值
		hexSub33 := func(hexStr string) string {
			value := new(big.Int)
			value.SetString(hexStr, 16)
			value.Sub(value, big.NewInt(0x33))
			return fmt.Sprintf("%02x", value)
		}

		dltDataFinished := hexSub33(newCommands[13])
		dltDataFinished1 := hexSub33(newCommands[12])
		dltDataFinished2 := hexSub33(newCommands[11])
		dltDataFinished3 := hexSub33(newCommands[10])

		makers := dltDataFinished + dltDataFinished1 + dltDataFinished2 + dltDataFinished3
		dlt.Debug("数据标识：%s", makers)

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
