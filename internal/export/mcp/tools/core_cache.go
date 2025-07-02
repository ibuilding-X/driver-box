package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/mark3labs/mcp-go/mcp"
)

var CoreCacheDevicesTool = mcp.NewTool("device_list",
	mcp.WithDescription("获取网关中的设备列表，返回结构化JSON数据，包含设备的唯一标识(id)、描述信息(description)、标签(tags)、属性(properties)、连接密钥(connectionKey)、驱动引用(driverKey)等完整设备信息。响应格式为Markdown，便于大模型处理和展示。"),
)

var CoreCacheDevicesHandler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	devices := helper.CoreCache.Devices()

	// 构建更适合大模型处理的结构化响应
	response := map[string]interface{}{
		"success": true,
		"data":    devices,
		"metadata": map[string]interface{}{
			"count": len(devices),
			"schema": map[string]interface{}{
				"device": map[string]string{
					"id":            "设备唯一标识符",
					"description":   "设备描述信息",
					"ttl":           "设备离线阈值",
					"tags":          "设备标签列表",
					"connectionKey": "连接密钥",
					"properties":    "设备属性映射",
					"driverKey":     "设备驱动引用",
				},
			},
			"format": "json",
		},
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError("序列化设备数据失败: " + err.Error()), err
	}

	// 转换为Markdown格式，更适合大模型处理
	markdown := "```json\n" + string(jsonData) + "\n```\n\n"
	markdown += fmt.Sprintf("共找到 %d 个设备\n", len(devices))

	return mcp.NewToolResultText(markdown), nil
}

var CoreCacheGetDeviceModelTool = mcp.NewTool("get_device_model",
	mcp.WithDescription("获取指定设备的物模型定义，返回结构化JSON数据，包含设备基本信息、模型信息及点位列表。点位信息包含点位ID、名称、类型、读写权限等完整定义。响应格式为Markdown，便于大模型理解和分析设备物模型结构。"),
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

	// 获取设备模型
	model, ok := helper.CoreCache.GetModel(device.ModelName)
	if !ok {
		return mcp.NewToolResultError(fmt.Sprintf("设备模型 %s 不存在", device.ModelName)), fmt.Errorf("设备模型不存在")
	}

	// 获取模型点位
	points, _ := helper.CoreCache.GetPoints(device.ModelName)

	// 构建更适合大模型处理的结构化响应
	response := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"device": device,
			"model":  model.ModelBase,
			"points": points,
		},
		"metadata": map[string]interface{}{
			"pointCount": len(points),
			"schema": map[string]interface{}{
				"point": map[string]string{
					"name":        "点位名称",
					"description": "点位描述",
					"valueType":   "数据类型(int/float/string)",
					"readWrite":   "读写类型(R/W/RW)",
					"reportMode":  "上报模式(realTime/change)",
				},
			},
			"format": "json",
		},
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError("序列化设备模型数据失败: " + err.Error()), err
	}

	// 转换为Markdown格式，更适合大模型处理
	markdown := "```json\n" + string(jsonData) + "\n```\n\n"
	markdown += fmt.Sprintf("设备 %s 的物模型定义，包含 %d 个点位\n", deviceID, len(points))

	return mcp.NewToolResultText(markdown), nil
}
