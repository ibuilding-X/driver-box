package history

import (
	"fmt"
	"strings"
)

type batchTableData struct {
	TableName  string
	SaveDbData []map[string]interface{}
}

type HistoryQueryParam struct {
	Conditions map[string]interface{}
	StartTime  string
	EndTime    string
	Columns    []interface{}
}

// 边缘计算查询
func (export0 *Export) QueryDataBySql(sql string) ([]map[string]interface{}, error) {

	rows, err := export0.db.Query(sql)
	if err != nil {
		fmt.Println("Database query failed: %v\n", err)
		return nil, err
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		fmt.Println("Database query result failed: %v\n", err)
		return nil, err
	}
	result := make([]map[string]interface{}, 0)
	for rows.Next() {
		row := make(map[string]interface{})
		values := make([]interface{}, len(columns))
		for i := range values {
			values[i] = new(interface{})
		}

		err := rows.Scan(values...)
		if err != nil {
			return nil, err
		}

		for i, colName := range columns {
			row[colName] = *(values[i].(*interface{}))
		}

		result = append(result, row)
	}

	return result, nil

}

// 历史数据查询
func (export0 *Export) QueryRealTimeData(params HistoryQueryParam) (map[string]interface{}, error) {
	return export0.queryDeviceData(params, TableNameSnapshotData)
}

func (export0 *Export) QueryDeviceHistoryData(params HistoryQueryParam) (map[string]interface{}, error) {
	return export0.queryDeviceData(params, TableNameRealTimeData)
}

func (export0 *Export) queryDeviceData(params HistoryQueryParam, tableName string) (map[string]interface{}, error) {

	if len(params.Conditions) < 1 {
		return nil, fmt.Errorf("query conditions is empty")
	}
	// 构建查询语句的基本部分
	columnNames := "id,create_time as timestamp,device_id as deviceId,mo_id as moId, "
	for _, columnName := range params.Columns {
		columnNames += fmt.Sprintf("json_extract(point_data,'$.%s') as %s, ", columnName, columnName)
	}
	columnNames = columnNames[:len(columnNames)-2]
	query := "SELECT " + columnNames + " FROM " + string(tableName)

	// 构建 WHERE 子句
	var conditions = make([]string, 0)
	var values = make([]interface{}, 0)
	for key, value := range params.Conditions {
		switch v := value.(type) {
		case string:
			conditions = append(conditions, fmt.Sprintf("%s = ?", key))
			values = append(values, v)
		case []string:
			// 构建 IN 子句，例如 moId IN ('v6', 'v8')
			placeholders := make([]string, len(v))
			for i := range v {
				placeholders[i] = "?"
				values = append(values, v[i])
			}
			conditions = append(conditions, fmt.Sprintf("%s IN (%s)", key, strings.Join(placeholders, ", ")))
		case []interface{}:
			// 构建 IN 子句，例如 moId IN ('v6', 'v8')
			placeholders := make([]string, len(v))
			for i := range v {
				placeholders[i] = "?"
				values = append(values, v[i])
			}
			conditions = append(conditions, fmt.Sprintf("%s IN (%s)", key, strings.Join(placeholders, ", ")))
		default:
			return nil, fmt.Errorf("unsupported condition type for key %s", key)
		}
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	// 添加时间范围条件
	if params.StartTime != "" && params.EndTime != "" {
		query += " AND create_time >= ? AND create_time <= ?"
		values = append(values, params.StartTime, params.EndTime)
	}

	// 准备查询语句
	stmt, err := export0.db.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	// 执行查询
	rows, err := stmt.Query(values...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// 构造结果集
	result := make([]map[string]interface{}, 0)
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		row := make(map[string]interface{})
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		for i, col := range columns {
			val := values[i]
			row[col] = val
		}

		result = append(result, row)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	deviceData := convertToMapOfSlices(result, "deviceId")
	return deviceData, nil

}

func convertToMapOfSlices(input []map[string]interface{}, key string) map[string]interface{} {
	response := make(map[string]interface{})
	result := make(map[string][]map[string]interface{})
	for _, item := range input {
		// 获取当前元素的键值
		value, ok := item[key].(string)
		if !ok {
			continue // 如果指定键不存在或者类型不符，跳过当前元素
		}

		// 在结果集中查找是否已经有对应键的切片，如果没有则初始化
		if _, exists := result[value]; !exists {
			result[value] = make([]map[string]interface{}, 0)
		}

		// 将当前元素添加到对应键的切片中
		result[value] = append(result[value], item)
	}
	for _k, _v := range result {
		response[_k] = _v
	}
	return response
}
