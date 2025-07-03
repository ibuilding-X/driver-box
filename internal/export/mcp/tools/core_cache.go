package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/mark3labs/mcp-go/mcp"
	"strings"
)

var CoreCacheDevicesTool = mcp.NewTool("device_list",
	mcp.WithDescription("获取网关中的设备列表，以表格形式展示设备的基本信息，包括设备ID、描述、模型名称、标签、属性等关键信息。响应格式为Markdown，便于直观阅读和理解。"),
)

var CoreCacheDevicesHandler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	devices := helper.CoreCache.Devices()

	// 构建表格形式的Markdown响应
	markdown := fmt.Sprintf("## 设备列表（共 %d 个设备）\n\n", len(devices))

	// 添加表格头部
	markdown += "| 设备ID | 描述 | 模型名称 | 标签 |  属性 |\n"
	markdown += "|---------|---------|---------|---------|---------|\n"

	// 添加表格内容
	for _, device := range devices {
		// 处理标签，将数组转换为逗号分隔的字符串
		tags := ""
		if len(device.Tags) > 0 {
			tagsBytes, err := json.Marshal(device.Tags)
			if err == nil {
				tags = string(tagsBytes)
				// 移除数组的方括号
				if len(tags) >= 2 {
					tags = tags[1 : len(tags)-1]
				}
				// 替换引号和逗号，使其更易读
				tags = strings.ReplaceAll(tags, "\"", "")
			}
		}

		properties := ""
		if len(device.Properties) > 0 {
			for k, v := range device.Properties {
				properties += fmt.Sprintf("%s : %s , ", k, v)
			}
		}
		properties = strings.ReplaceAll(properties, "|", "\\|")

		// 处理描述中可能存在的特殊字符，避免破坏Markdown表格结构
		description := strings.ReplaceAll(device.Description, "|", "\\|")
		description = strings.ReplaceAll(description, "\n", " ")

		// 添加设备行
		markdown += fmt.Sprintf("| %s | %s | %s | %s | %s |\n",
			device.ID,
			description,
			device.ModelName,
			tags,
			properties)
	}

	return mcp.NewToolResultText(markdown), nil
}

var CoreCacheGetDeviceModelTool = mcp.NewTool("get_device_model",
	mcp.WithDescription("获取指定设备的物模型定义，以表格形式展示设备基本信息和点位列表。设备基本信息包括设备ID、描述、模型名称、驱动和离线阈值；点位列表包含点位名称、描述、数据类型、读写类型和上报模式。"),
	mcp.WithString("id", mcp.Required(), mcp.Description("设备唯一标识符，用于查询特定设备的物模型信息")),
)

var CoreCacheGetDeviceModelHandler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 获取设备ID参数
	deviceID := request.GetString("id", "")
	if deviceID == "" {
		return mcp.NewToolResultError("设备ID不能为空"), fmt.Errorf("设备ID不能为空")
	}

	// 获取设备信息
	device, ok := helper.CoreCache.GetDevice(deviceID)
	if !ok {
		return mcp.NewToolResultError(fmt.Sprintf("设备 %s 不存在", deviceID)), fmt.Errorf("设备不存在")
	}

	// 获取模型点位
	points, _ := helper.CoreCache.GetPoints(device.ModelName)

	// 构建表格形式的Markdown响应
	markdown := fmt.Sprintf("## 设备 `%s` 的物模型\n\n", deviceID)

	// 添加设备基本信息
	markdown += "### 设备基本信息\n\n"
	markdown += "| 属性 | 值 |\n"
	markdown += "|---------|---------|\n"

	// 处理描述中可能存在的特殊字符，避免破坏Markdown表格结构
	description := strings.ReplaceAll(device.Description, "|", "\\|")
	description = strings.ReplaceAll(description, "\n", " ")

	// 处理其他字段中可能存在的特殊字符
	modelName := strings.ReplaceAll(device.ModelName, "|", "\\|")
	driverKey := strings.ReplaceAll(device.DriverKey, "|", "\\|")

	markdown += fmt.Sprintf("| 设备ID | %s |\n", device.ID)
	markdown += fmt.Sprintf("| 设备描述 | %s |\n", description)
	markdown += fmt.Sprintf("| 模型名称 | %s |\n", modelName)
	markdown += fmt.Sprintf("| 驱动 | %s |\n", driverKey)
	markdown += fmt.Sprintf("| 离线阈值 | %s |\n", device.Ttl)

	// 添加点位列表
	markdown += fmt.Sprintf("\n### 点位列表（共 %d 个点位）\n\n", len(points))
	markdown += "| 点位名称 | 描述 | 数据类型 | 读写类型 | 上报模式 |\n"
	markdown += "|---------|---------|---------|---------|---------|\n"

	// 添加点位表格内容
	for _, point := range points {
		// 处理描述中可能存在的特殊字符，避免破坏Markdown表格结构
		description := strings.ReplaceAll(point.Description, "|", "\\|")
		description = strings.ReplaceAll(description, "\n", " ")

		// 处理其他字段中可能存在的特殊字符
		valueType := strings.ReplaceAll(string(point.ValueType), "|", "\\|")
		readWrite := strings.ReplaceAll(string(point.ReadWrite), "|", "\\|")
		reportMode := strings.ReplaceAll(string(point.ReportMode), "|", "\\|")

		markdown += fmt.Sprintf("| `%s` | %s | %s | %s | %s |\n",
			point.Name,
			description,
			valueType,
			readWrite,
			reportMode)
	}

	return mcp.NewToolResultText(markdown), nil
}
