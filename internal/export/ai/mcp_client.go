package ai

import (
	langchaingo_mcp_adapter "github.com/i2y/langchaingo-mcp-adapter"
	"github.com/ibuilding-x/driver-box/driverbox"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/tmc/langchaingo/tools"
	"go.uber.org/zap"
)

func (export *Export) getTools() ([]mcp.Tool, error) {
	// sse client
	cli, err := client.NewStreamableHttpClient("http://localhost:8999/mcp")
	// sse client  needs to manually start asynchronous communication
	// while stdio does not require it.
	err = cli.Start(export.ctx)

	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "driver-box-client",
		Version: "1.0.0",
	}
	initRequest.Params.Capabilities = mcp.ClientCapabilities{}

	_, err = cli.Initialize(export.ctx, initRequest)
	if err != nil {
		helper.Logger.Error("initialize error", zap.Error(err))
	}
	r, e := cli.ListTools(export.ctx, mcp.ListToolsRequest{})
	if e != nil {
		return nil, e
	}
	return r.Tools, nil

}

func (export *Export) getLangChainTools() ([]tools.Tool, error) {
	// Create an MCP client using stdio
	// sse client
	cli, err := client.NewSSEMCPClient("http://localhost:8999/sse")
	// sse client  needs to manually start asynchronous communication
	// while stdio does not require it.
	err = cli.Start(export.ctx)
	//defer cli.Close()

	// Create the adapter
	adapter, err := langchaingo_mcp_adapter.New(cli)
	if err != nil {
		helper.Logger.Error("Failed to create adapter", zap.Error(err))
		return nil, err
	}

	// Get all tools from MCP server
	return adapter.Tools()
}
