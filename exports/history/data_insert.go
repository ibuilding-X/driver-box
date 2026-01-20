package history

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/ibuilding-x/driver-box/driverbox"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"go.uber.org/zap"
)

type deviceQueueData struct {
	deviceData plugin.DeviceData
	addTime    time.Time
}

func (export0 *Export) writeRealTimeData(queue []deviceQueueData) {
	num := len(queue)
	batchRealTimeData := batchTableData{
		TableName:  TableNameRealTimeData,
		SaveDbData: make([]map[string]interface{}, 0, num),
	}

	for _, deviceData := range queue {
		deviceId := deviceData.deviceData.ID
		device, res := driverbox.CoreCache().GetDevice(deviceId)
		pointData := map[string]interface{}{
			"device_id":   deviceId,
			"create_time": deviceData.addTime,
		}
		if res {
			model, success := driverbox.CoreCache().GetModel(device.ModelName)
			if success {
				pointData["mo_id"] = model.ModelID
			}
		}
		pointNum := len(deviceData.deviceData.Values)
		if pointNum > 0 {
			dataPoint := make(map[string]interface{}, pointNum)
			for _, item := range deviceData.deviceData.Values {
				dataPoint[item.PointName] = item.Value
			}
			pointStr, er := json.Marshal(dataPoint)
			if er != nil {
				helper.Logger.Error(fmt.Sprintf("realTime points json marshal error %v", er.Error()))
			}
			pointData["point_data"] = string(pointStr)
		}
		batchRealTimeData.SaveDbData = append(batchRealTimeData.SaveDbData, pointData)
	}
	export0.batchInsert(batchRealTimeData)
}

func (export0 *Export) writeDeviceSnapshotData() {

	// 1. 获取设备影子数据 填充
	devices := driverbox.Shadow().GetDevices()
	// 2. 设备数据批量落库，50台设备一批次
	total, pageSize := len(devices), 50
	pageIndex := int(math.Ceil(float64(total) / float64(pageSize)))
	startRow := 0
	now := time.Now()
	for i := 1; i <= pageIndex; i++ {
		batchHistoryData := batchTableData{
			TableName:  TableNameSnapshotData,
			SaveDbData: make([]map[string]interface{}, 0, pageSize),
		}

		maxIdx := i * pageSize
		if maxIdx > total {
			maxIdx = total
		}
		for j := startRow; j < maxIdx; j++ {
			device := devices[j]
			modelName := device.ModelName
			model, res := driverbox.CoreCache().GetModel(modelName)
			pointData := map[string]interface{}{
				"device_id":   device.ID,
				"create_time": now,
			}
			if res {
				pointData["mo_id"] = model.ModelID
			}
			if len(device.Points) > 0 {
				dataPoint := make(map[string]interface{})
				for _, item := range device.Points {
					dataPoint[item.Name] = item.Value
				}
				pointStr, er := json.Marshal(dataPoint)
				if er != nil {
					helper.Logger.Error(fmt.Sprintf("realTime points json marshal error %v", er.Error()))
					continue
				}
				pointData["point_data"] = string(pointStr)
			} else {
				pointData["point_data"] = "{}"
				//helper.Logger.Info(fmt.Sprintf("device points is empty, device id is %v", device.ID))
			}

			//记录设备的metadata
			meta := make(map[string]interface{})
			if device.Online {
				meta["online"] = 1
			} else {
				meta["online"] = 0
			}

			str, er := json.Marshal(meta)
			if er != nil {
				helper.Logger.Error("meta json marshal error", zap.Any("meta", meta), zap.Error(er))
			} else {
				pointData["meta"] = str
			}
			batchHistoryData.SaveDbData = append(batchHistoryData.SaveDbData, pointData)
		}
		startRow += pageSize
		export0.batchInsert(batchHistoryData)
	}

}

// 历史数据写入数据
func (export0 *Export) batchInsert(data batchTableData) {
	if len(data.SaveDbData) == 0 {
		// 如果没有数据需要保存，直接返回
		return
	}

	var (
		columnNames       []string
		valuePlaceholders []string
		params            []interface{}
	)

	// 构建列名和值占位符
	for idx, rowData := range data.SaveDbData {
		var columnValues []interface{}

		if idx == 0 {
			for columnName, _ := range rowData {
				columnNames = append(columnNames, columnName)
			}
		}
		for _, key := range columnNames {
			columnValues = append(columnValues, rowData[key])
		}

		valuePlaceholderList := make([]string, 0)
		for range columnValues {
			valuePlaceholderList = append(valuePlaceholderList, "?")
		}
		valuePlaceholderStr := "(" + strings.Join(valuePlaceholderList, ",") + ")"
		valuePlaceholders = append(valuePlaceholders, valuePlaceholderStr)
		params = append(params, columnValues...)
	}

	columnNamesStr := strings.Join(columnNames, ", ")
	valuesStr := strings.Join(valuePlaceholders, ", ")

	query := fmt.Sprintf("INSERT OR REPLACE INTO %s (%s) VALUES %s", data.TableName, columnNamesStr, valuesStr)

	// 执行 SQL 查询
	result, err := export0.db.Exec(query, params...)
	if err != nil {
		helper.Logger.Error(err.Error())
		return
	}

	rowsAffected, _ := result.RowsAffected()
	helper.Logger.Info(fmt.Sprintf("%v data batch insert successful:%d rows", data.TableName, rowsAffected))

}
