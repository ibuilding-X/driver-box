package tools

import (
	"github.com/mark3labs/mcp-go/mcp"
)

var WritePointsTool = mcp.NewTool("write_points",
	mcp.WithDescription("针对指定设备下发控制指令"),
	mcp.WithString("id", mcp.Required(), mcp.Description("设备ID")),
	mcp.WithArray("points", mcp.Required(), mcp.Description("设备点信息")),
	mcp.WithObject("points",
		mcp.Required(),
		mcp.Description("设备点信息"),
		mcp.Properties(map[string]any{
			"name":  map[string]any{"type": "string"},
			"value": map[string]any{"type": "string"},
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
