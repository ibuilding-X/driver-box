package tools

import (
	"context"
	"encoding/json"

	"github.com/ibuilding-x/driver-box/v2/driverbox"
	"github.com/mark3labs/mcp-go/mcp"
)

var ShadowDeviceListTool = mcp.NewTool("device_shadow_list",
	mcp.WithDescription("获取网关中的所有设备影子数据，返回JSON格式的设备影子列表。设备影子是设备状态的虚拟表示，包含设备的最新状态信息、点位值和连接状态等数据，可用于了解设备的实时状态。"),
)

var ShadowDeviceListHandler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	devices := driverbox.Shadow().GetDevices()
	jsonData, _ := json.Marshal(devices)
	return mcp.NewToolResultText(string(jsonData)), nil
}

var ShadowDeviceTool = mcp.NewTool("device_shadow_info",
	mcp.WithDescription("获取网关中指定设备ID的影子数据，返回JSON格式的单个设备影子信息。通过设备ID可以查询特定设备的实时状态、点位值和连接状态等详细信息，便于针对性地分析和监控设备。"),
	mcp.WithString("id", mcp.Required(), mcp.Description("设备唯一标识符，用于查询特定设备的影子数据")),
)

var ShadowDeviceHandler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := request.RequireString("id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	devices, _ := driverbox.Shadow().GetDevice(id)
	jsonData, _ := json.Marshal(devices)
	return mcp.NewToolResultText(string(jsonData)), nil
}
