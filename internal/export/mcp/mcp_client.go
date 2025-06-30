package mcp

import (
	"context"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/mark3labs/mcp-go/mcp"
	"go.uber.org/zap"
	"time"

	emcp "github.com/cloudwego/eino-ext/components/tool/mcp"
	"github.com/cloudwego/eino/components/tool"
	"github.com/mark3labs/mcp-go/client"
)

func (export *Export) getTools() ([]tool.BaseTool, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	// sse client
	cli, err := client.NewStreamableHttpClient("http://localhost:8999/mcp")
	// sse client  needs to manually start asynchronous communication
	// while stdio does not require it.
	err = cli.Start(ctx)

	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "driver-box-client",
		Version: "1.0.0",
	}
	initRequest.Params.Capabilities = mcp.ClientCapabilities{}

	_, err = cli.Initialize(ctx, initRequest)
	if err != nil {
		helper.Logger.Error("initialize error", zap.Error(err))
	}
	return emcp.GetTools(ctx, &emcp.Config{Cli: cli})

}
