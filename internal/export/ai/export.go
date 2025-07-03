package ai

import (
	"context"
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	tools2 "github.com/ibuilding-x/driver-box/internal/export/ai/mcp/tools"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
	"go.uber.org/zap"
	"net/http"
	"os"
	"sync"
	"time"
)

var driverInstance *Export
var once = &sync.Once{}

type Export struct {
	baseUrl    string
	model      string
	ready      bool
	mcpServers []*server.MCPServer
	ctx        context.Context
	llm        llms.Model
}

func (export *Export) Init() error {
	if os.Getenv(config.ENV_EXPORT_MCP_ENABLED) == "false" {
		helper.Logger.Warn("mcp export is disabled")
		return nil
	}
	go func() {
		e := export.run("sse", ":8999")
		if e != nil {
			export.ready = false
		}
	}()

	llm, err := ollama.New(ollama.WithModel(export.model), ollama.WithServerURL(export.baseUrl))
	if err != nil {
		return err
	}
	export.llm = llm
	go func() {
		time.Sleep(1 * time.Second)
		export.startAgent()
	}()

	export.ready = true
	return nil
}
func NewExport() *Export {
	once.Do(func() {
		driverInstance = &Export{
			ctx:     context.Background(),
			model:   "qwen3:8b",
			baseUrl: "http://192.168.16.94:11434",
		}
	})
	return driverInstance
}

// 点位变化触发场景联动
func (export *Export) ExportTo(deviceData plugin.DeviceData) {

}

// 继承Export OnEvent接口
func (export *Export) OnEvent(eventCode string, key string, eventValue interface{}) error {
	return nil
}

func (export *Export) IsReady() bool {
	return export.ready
}

func (export *Export) addTools(s *server.MCPServer) {
	// Repository Tools
	s.AddTool(tools2.CoreCacheDevicesTool, tools2.CoreCacheDevicesHandler)
	s.AddTool(tools2.CoreCacheGetModelByDeviceTool, tools2.CoreCacheGetModelByDeviceHandler)
	s.AddTool(tools2.CoreCacheGetModelByNameTool, tools2.CoreCacheGetModelByNameHandler)
	s.AddTool(tools2.ShadowDeviceListTool, tools2.ShadowDeviceListHandler)
	s.AddTool(tools2.ShadowDeviceTool, tools2.ShadowDeviceHandler)
	s.AddTool(tools2.WritePointsTool, tools2.WritePointsHandler)
	s.AddTool(tools2.HistoryTableSchemaTool, tools2.HistoryTableSchemaHandler)
	s.AddTool(tools2.HistoryDataAnalysisTool, tools2.HistoryDataAnalysisHandler)
}

func (export *Export) newMCPServer() *server.MCPServer {
	hooks := &server.Hooks{}

	hooks.OnBeforeCallTool = append(hooks.OnBeforeCallTool, func(ctx context.Context, id any, message *mcp.CallToolRequest) {
		helper.Logger.Info("[tool call]", zap.String("ToolName", message.Params.Name), zap.Any("Params", message.Params.Arguments))
	})

	hooks.OnAfterCallTool = append(hooks.OnAfterCallTool, func(ctx context.Context, id any, message *mcp.CallToolRequest, result *mcp.CallToolResult) {
		if result != nil && result.IsError {
			helper.Logger.Error("[tool call error]", zap.String("ToolName", message.Params.Name), zap.Any("Params", message.Params.Arguments), zap.Any("error", result.Content))
		}
	})

	return server.NewMCPServer(
		"driver-box",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithLogging(),
		server.WithHooks(hooks),
	)
}

func (export *Export) run(transport, addr string) error {
	s := export.newMCPServer()
	export.addTools(s)

	switch transport {
	case "stdio":
		if err := server.ServeStdio(s); err != nil {
			if err == context.Canceled {
				return nil
			}
			return err
		}
	case "sse":
		srv := server.NewSSEServer(s, server.WithBaseURL(addr))
		if err := srv.Start(addr); err != nil {
			helper.Logger.Error("start sse server error", zap.Error(err))
			if err == context.Canceled {
				return nil
			}
			return fmt.Errorf("server error: %v", err)
		}
	case "http":
		httpServer := server.NewStreamableHTTPServer(s,
			server.WithStateLess(true),
			server.WithHTTPContextFunc(func(ctx context.Context, r *http.Request) context.Context {
				auth := r.Header.Get("Authorization")
				if len(auth) > 7 && auth[:7] == "Bearer " {
					token := auth[7:]
					ctx = context.WithValue(ctx, "access_token", token)
				}
				return ctx
			}),
		)
		if err := httpServer.Start(addr); err != nil {
			if err == context.Canceled {
				return nil
			}
			return fmt.Errorf("server error: %v", err)
		}
	default:
		return fmt.Errorf(
			"invalid transport type: %s. Must be 'stdio'、'sse' or 'http'",
			transport,
		)
	}
	return nil
}
