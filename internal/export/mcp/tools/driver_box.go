package tools

import (
	"github.com/mark3labs/mcp-go/mcp"
)

var WritePointsTool = mcp.NewTool("write_points",
	mcp.WithDescription("针对指定设备下发控制指令，用于修改设备点位的值。通过提供设备ID和需要修改的点位列表，可以实现对设备的远程控制。每个点位需要指定名称和要设置的值。"),
	mcp.WithString("id", mcp.Required(), mcp.Description("设备唯一标识符，用于指定要控制的目标设备")),
	mcp.WithArray("points", mcp.Required(), mcp.Description("需要控制的设备点位列表，每个点位包含名称和值")),
	mcp.WithObject("points",
		mcp.Required(),
		mcp.Description("设备点位信息结构，定义了点位的名称和要设置的值"),
		mcp.Properties(map[string]any{
			"name":  map[string]any{"type": "string", "description": "点位名称，必须与设备物模型中定义的点位名称一致"},
			"value": map[string]any{"type": "string", "description": "要设置的点位值，数据类型需与点位定义一致"},
		}),
	),
)

//var WritePointsHandler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
//	id, err := request.RequireString("id")
//	if err != nil {
//		return mcp.NewToolResultError(err.Error()), nil
//	}
//	request.RequireString()
//	core.SendBatchWrite()
//	devices := helper.DeviceShadow.GetDevices()
//	jsonData, _ := json.Marshal(devices)
//	return mcp.NewToolResultText(string(jsonData)), nil
//}
