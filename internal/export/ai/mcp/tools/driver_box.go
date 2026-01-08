package tools

import (
	"context"
	"fmt"
	"github.com/ibuilding-x/driver-box/internal/core"
	"github.com/ibuilding-x/driver-box/pkg/driverbox/plugin"
	"github.com/mark3labs/mcp-go/mcp"
)

var WritePointsTool = mcp.NewTool("write_points",
	mcp.WithDescription("针对指定设备下发控制指令，用于修改设备点位的值。通过提供设备ID和需要修改的点位列表，可以实现对设备的远程控制。"),
	mcp.WithString("device_id", mcp.Required(), mcp.Description("设备唯一标识符，用于指定要控制的目标设备")),
	mcp.WithArray("points",
		mcp.Required(),
		mcp.Description("设备点位信息结构，定义了点位的名称和要设置的值"),
		mcp.Properties(map[string]any{
			"name":  map[string]any{"type": "string", "description": "点位名称，必须与设备物模型中定义的点位名称一致"},
			"value": map[string]any{"type": "any", "description": "要设置的点位值，支持多种数据类型(字符串、数字、布尔值等)，需与点位定义的类型一致"},
		}),
	),
)

var WritePointsHandler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 获取设备ID
	deviceID, err := request.RequireString("device_id")
	if err != nil {
		return mcp.NewToolResultError("设备ID参数错误: " + err.Error()), nil
	}

	// 获取点位列表
	if request.GetArguments() == nil {
		return mcp.NewToolResultError("点位列表参数错误"), nil
	}
	args := request.GetArguments()["points"]

	// 转换为PointData结构
	points := make([]plugin.PointData, 0)
	for _, v := range args.([]any) {
		v := v.(map[string]any)
		points = append(points, plugin.PointData{
			PointName: v["name"].(string),
			Value:     v["value"],
		})
	}

	// 执行点位写入
	err = core.SendBatchWrite(deviceID, points)
	if err != nil {
		return mcp.NewToolResultError("写入点位失败: " + err.Error()), err
	}

	return mcp.NewToolResultText("成功写入 " + deviceID + " 设备的 " + fmt.Sprintf("%d", len(points)) + " 个点位"), nil
}
