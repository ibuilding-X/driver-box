package dlt645

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"
)

type Dlt645ConfigClient struct {
	MeterNumber string
	DataMarker  string
}

func (dltconfig *Dlt645ConfigClient) SendMessageToSerial(dlt Client) (response float64, err error) {
	//表号
	meterNumberHandle := HexStringToBytes(dltconfig.MeterNumber)
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
	DataMarkerHandle := HexStringToBytes(dltconfig.DataMarker)
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
	value, err := dlt.SendRawFrame(CheckCode(messageFinshed))
	return value, err
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
