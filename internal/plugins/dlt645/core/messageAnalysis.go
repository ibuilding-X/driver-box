package dlt645

import (
	"errors"
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/shopspring/decimal"
	"strconv"
	"strings"
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

func analysis(dlt *Dlt645ClientProvider, command string) float64 {
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
		return 0
	} else {
		helper.Logger.Debug(fmt.Sprintf("报文源码：%s", command))
		helper.Logger.Debug(fmt.Sprintf("帧起始符：%s", newCommands[0]))
		helper.Logger.Debug(fmt.Sprintf("报文源码：%s", command))
		helper.Logger.Debug(fmt.Sprintf("报文源码：%s", command))
		helper.Logger.Debug(fmt.Sprintf("报文源码：%s", command))
		dlt.Debug("帧起始符：%s", newCommands[0])
		meter_serial := newCommands[6] + newCommands[5] + newCommands[4] + newCommands[3] + newCommands[2] + newCommands[1]
		dlt.Debug("电表地址：%s", meter_serial)
		dlt.Debug("控制域：%s", newCommands[8])
		dlt.Debug("数据域长度：%s", newCommands[9])
		dlt.Debug("校验码：%s", newCommands[len(newCommands)-2])
		dlt.Debug("停止位：%s", newCommands[len(newCommands)-1])

		dltData, _ := strconv.ParseUint(newCommands[13], 16, 32)
		dltData = dltData - 51
		var dltDataFinished string
		if len(strconv.FormatInt(int64(dltData), 10)) == 1 {
			dltDataFinished = "0" + strconv.Itoa(int(dltData))
		}

		dltData1, _ := strconv.ParseUint(newCommands[12], 16, 32)
		dltData1 = dltData1 - 51
		var dltDataFinished1 string
		if len(strconv.FormatInt(int64(dltData1), 10)) == 1 {
			dltDataFinished1 = "0" + strconv.Itoa(int(dltData1))
		}

		dltData2, _ := strconv.ParseUint(newCommands[11], 16, 32)
		dltData2 = dltData2 - 51
		var dltDataFinished2 string

		if len(strconv.FormatInt(int64(dltData2), 10)) == 1 {
			dltDataFinished2 = "0" + strconv.Itoa(int(dltData2))
		}

		dltData3, _ := strconv.ParseUint(newCommands[10], 16, 32)
		dltData3 = dltData3 - 51

		var dltDataFinished3 string
		if len(strconv.FormatInt(int64(dltData3), 10)) == 1 {
			dltDataFinished3 = "0" + strconv.Itoa(int(dltData3))
		}

		makers := dltDataFinished + dltDataFinished1 + dltDataFinished2 + dltDataFinished3
		dlt.Debug("数据标识：%s", makers)
		dataUnits := len(newCommands) - 2
		var data string
		for i := dataUnits; i > 14; i-- {
			v1, _ := strconv.ParseUint(newCommands[i-1], 16, 32)
			v2, _ := strconv.ParseUint("33", 16, 32)
			midData, _ := DecConvertToX(int(v1-v2), 16)
			if len(midData) == 1 {
				midData = "0" + midData
			}
			data += fmt.Sprintf("%s", midData)
		}

		// 原始值
		n1, _ := decimal.NewFromString(data)
		// 系数
		var n2 decimal.Decimal

		if dltDataFinished == "00" {
			n2, _ = decimal.NewFromString("0.01")
		} else if dltDataFinished == "02" && dltDataFinished1 == "01" {
			n2, _ = decimal.NewFromString("0.1")
		} else if dltDataFinished == "02" && dltDataFinished1 == "02" {
			n2, _ = decimal.NewFromString("0.01")
		} else if (dltDataFinished == "02" && dltDataFinished1 == "03") || (dltDataFinished == "02" && dltDataFinished1 == "04") || (dltDataFinished == "02" && dltDataFinished1 == "05") {
			n2, _ = decimal.NewFromString("0.0001")
		} else if dltDataFinished == "02" && dltDataFinished1 == "06" {
			n2, _ = decimal.NewFromString("0.001")
		} else if dltDataFinished == "02" && dltDataFinished3 == "02" {
			n2, _ = decimal.NewFromString("0.01")
		} else if dltDataFinished == "02" && dltDataFinished1 == "01" {
			n2, _ = decimal.NewFromString("0.1")
		} else if dltDataFinished == "02" && dltDataFinished1 == "02" {
			n2, _ = decimal.NewFromString("0.001")
		}

		over := n1.Mul(n2)
		val, b := over.Float64()
		if !b {

		}
		return val
	}
	return 0
}
