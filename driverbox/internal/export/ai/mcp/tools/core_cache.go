package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/mark3labs/mcp-go/mcp"
)

var CoreCacheDevicesTool = mcp.NewTool("device_list",
	mcp.WithDescription("获取网关中的设备列表，以表格形式展示设备的基本信息，包括设备ID、描述、模型ID、标签、属性等关键信息。响应格式为Markdown，便于直观阅读和理解。"),
)

var CoreCacheDevicesHandler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	devices := helper.CoreCache.Devices()

	// 构建表格形式的Markdown响应
	markdown := fmt.Sprintf("## 设备列表（共 %d 个设备）\n\n", len(devices))

	// 添加表格头部
	markdown += "| 设备ID | 描述 | 模型ID | 标签 |  属性 |\n"
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

var CoreCacheGetModelByDeviceTool = mcp.NewTool("get_model_by_deviceId",
	mcp.WithDescription("获取指定设备的物模型定义，以表格形式展示设备基本信息和点位列表。设备基本信息包括设备ID、描述、模型名称、驱动和离线阈值；点位列表包含点位名称、描述、数据类型、读写类型和上报模式。"),
	mcp.WithString("deviceId", mcp.Required(), mcp.Description("设备唯一标识符，用于查询特定设备的物模型信息")),
)

var CoreCacheGetModelByDeviceHandler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 获取设备ID参数
	deviceID := request.GetString("deviceId", "")
	if deviceID == "" {
		return mcp.NewToolResultError("设备ID不能为空"), fmt.Errorf("设备ID不能为空")
	}

	// 获取设备信息
	device, ok := helper.CoreCache.GetDevice(deviceID)
	if !ok {
		return mcp.NewToolResultError(fmt.Sprintf("设备 %s 不存在", deviceID)), fmt.Errorf("设备不存在")
	}
	result, err := getModelInfo(device.ModelName)
	return mcp.NewToolResultText(result), err
}

var CoreCacheGetModelByNameTool = mcp.NewTool("get_model_by_modelName",
	mcp.WithDescription("获取指定模型名称的物模型定义，以表格形式展示模型基本信息和点位列表。点位列表包含点位名称、描述、数据类型、读写类型和上报模式。"),
	mcp.WithString("modelName", mcp.Required(), mcp.Description("模型唯一标识符，用于查询特定modelName的物模型信息")),
)

var CoreCacheGetModelByNameHandler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 获取设备ID参数
	modelName := request.GetString("modelName", "")
	result, err := getModelInfo(modelName)
	return mcp.NewToolResultText(result), err
}

func getModelInfo(modelName string) (string, error) {
	model, ok := helper.CoreCache.GetModel(modelName)
	if !ok {
		return "", errors.New("模型不存在")
	}
	markdown := fmt.Sprintf("## 模型 `%s` 的基本信息\n\n", modelName)
	markdown += "模型名称：" + model.Name
	markdown += "\n模型ID：" + model.ModelID
	markdown += "\n模型描述：" + model.Description
	markdown += "\n模型属性："
	for k, v := range model.Attributes {
		markdown += fmt.Sprintf("\n\t%s：%s", k, v)
	}
	markdown += fmt.Sprintf("\n### 点位列表（共 %d 个点位）\n\n", len(model.DevicePoints))
	markdown += "| 点位名称 | 描述 | 数据类型 | 读写类型 | 上报模式 | 点值枚举表 |\n"
	markdown += "|---------|---------|---------|---------|---------|---------|\n"
	for _, point := range model.DevicePoints {
		enums := "<unknow>"
		if len(point.Enums()) > 0 {
			bytes, _ := json.Marshal(point.Enums())
			enums = string(bytes)
		}
		markdown += fmt.Sprintf("| %s | %s | %s | %s | %s | %s |\n", point.Name, point.Description, point.ValueType, point.ReadWrite, point.ReportMode, enums)
	}
	return markdown, nil
}
