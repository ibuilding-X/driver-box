package mcp

import (
	"context"
	"fmt"
	"github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"go.uber.org/zap"
)

func (export *Export) startAgent() error {
	mcpTools, e := export.getTools()
	if e != nil {
		return e
	}
	// 初始化 tools

	todoTools := make([]tool.BaseTool, 0)
	todoTools = append(todoTools, mcpTools...)

	// 创建并配置 ChatModel
	chatModel, err := ollama.NewChatModel(context.Background(), &ollama.ChatModelConfig{
		BaseURL: "http://192.168.16.94:11434",
		Model:   "qwen3:8b",
	})
	if err != nil {
		helper.Logger.Error("init chat model error", zap.Error(err))
	}
	// 获取工具信息并绑定到 ChatModel
	toolInfos := make([]*schema.ToolInfo, 0, len(mcpTools))
	for _, tool := range todoTools {
		info, err := tool.Info(export.ctx)
		if err != nil {
			helper.Logger.Error("get tool info error", zap.Error(err))
		}
		toolInfos = append(toolInfos, info)
	}
	err = chatModel.BindTools(toolInfos)
	if err != nil {
		helper.Logger.Error("bind tools to chat model error", zap.Error(err))
	}

	// 创建 tools 节点
	todoToolsNode, err := compose.NewToolNode(context.Background(), &compose.ToolsNodeConfig{
		Tools: todoTools,
	})
	if err != nil {
		helper.Logger.Error("new tool node error", zap.Error(err))
	}

	// 构建完整的处理链
	chain := compose.NewChain[[]*schema.Message, []*schema.Message]()
	chain.
		AppendChatModel(chatModel, compose.WithNodeName("chat_model")).
		AppendToolsNode(todoToolsNode, compose.WithNodeName("tools"))

	// 编译并运行 chain
	agent, err := chain.Compile(export.ctx)
	if err != nil {
		helper.Logger.Error("init chain error", zap.Error(err))
	}

	// 运行示例
	resp, err := agent.Invoke(export.ctx, []*schema.Message{
		schema.SystemMessage("你是一名专业的设备管理专家"),
		{
			Role:    schema.User,
			Content: "这个网关当前有多少设备",
		},
	})
	if err != nil {
		helper.Logger.Error("invoke chain error", zap.Error(err))
		return err
	}

	// 输出结果
	for _, msg := range resp {
		fmt.Println(msg.Content)
	}
	return nil
}
