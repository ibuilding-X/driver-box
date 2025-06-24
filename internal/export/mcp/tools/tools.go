package tools

import (
	"context"
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/mark3labs/mcp-go/mcp"
)

var CoreCacheDevicesTool = mcp.NewTool("devices",
	mcp.WithDescription("获取设备列表"),
)

var CoreCacheDevicesHandler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {

	devices := helper.CoreCache.Devices()
	var result = ""
	for _, device := range devices {
		result = result + fmt.Sprintf("设备ID：%s, 设备名称：%s,  设备标签：%s, 设备模型：%s, 设备属性：%s\n", device.ID, device.Description, device.Tags, device.ModelName, device.Properties)
	}

	return mcp.NewToolResultText(result), nil
}
