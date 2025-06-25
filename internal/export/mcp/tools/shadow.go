package tools

import (
	"context"
	"encoding/json"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/mark3labs/mcp-go/mcp"
)

var ShadowDeviceListTool = mcp.NewTool("device_shadow_list",
	mcp.WithDescription("获取网关中的所有设备影子数据"),
)

var ShadowDeviceListHandler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	devices := helper.DeviceShadow.GetDevices()
	jsonData, _ := json.Marshal(devices)
	return mcp.NewToolResultText(string(jsonData)), nil
}

var ShadowDeviceTool = mcp.NewTool("device_shadow_info",
	mcp.WithDescription("获取网关中指定设备ID的影子数据"),
	mcp.WithString("id", mcp.Required()),
)

var ShadowDeviceHandler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := request.RequireString("id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	devices, _ := helper.DeviceShadow.GetDevice(id)
	jsonData, _ := json.Marshal(devices)
	return mcp.NewToolResultText(string(jsonData)), nil
}
