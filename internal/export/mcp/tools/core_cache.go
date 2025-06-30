package tools

import (
	"context"
	"encoding/json"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/mark3labs/mcp-go/mcp"
)

var CoreCacheDevicesTool = mcp.NewTool("device_list",
	mcp.WithDescription("获取网关中的设备列表"),
)

var CoreCacheDevicesHandler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	devices := helper.CoreCache.Devices()
	jsonData, _ := json.Marshal(devices)
	return mcp.NewToolResultText(string(jsonData)), nil
}

var CoreCacheGetDeviceModelTool = mcp.NewTool("get_device_model",
	mcp.WithDescription("获取指定设备的物模型定义"),
	mcp.WithString("id", mcp.Required(), mcp.Description("设备ID")),
)
