package tools

import (
	"context"
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/export/history"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/mark3labs/mcp-go/mcp"
	"go.uber.org/zap"
)

var HistoryTableSchemaTool = mcp.NewTool("history_table_schema",
	mcp.WithDescription("查询当前网关数据库的表结构定义"),
)

var HistoryTableSchemaHandler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	r, e := history.NewExport().QueryDataBySql("SELECT name,sql FROM sqlite_master WHERE type='table';")
	helper.Logger.Info("查询当前网关数据库的表结构定义", zap.Any("result", r), zap.Error(e))
	if e != nil {
		return nil, e
	}
	markdown := fmt.Sprintf("## 表结构定义（共 %d 张表）\n\n", len(r))
	for _, v := range r {
		markdown += fmt.Sprintf("### tableName: %s\n\n```sql\n%s\n```\n\n", v["name"], v["sql"])
	}
	return mcp.NewToolResultText(markdown), nil
}
