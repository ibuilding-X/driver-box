package mcp

import (
	"context"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms/ollama"
	"go.uber.org/zap"
)

func (export *Export) startAgent() error {
	tools, e := export.getLangChainTools()

	if e != nil {
		return e
	}
	llm, err := ollama.New(ollama.WithModel(export.model), ollama.WithServerURL(export.baseUrl))
	if err != nil {
		return err
	}
	ctx := context.Background()

	agent := agents.NewOneShotAgent(
		llm,
		tools,
		agents.WithMaxIterations(3),
	)
	executor := agents.NewExecutor(agent)
	// Use the agent
	question := "网关中有多少类设备，没类设备有多少个？"
	result, err := chains.Run(
		ctx,
		executor,
		question,
	)
	helper.Logger.Info("", zap.Any("result", result))

	//messages := make([]llms.MessageContent, 0)
	//messages = append(messages, llms.TextParts(llms.ChatMessageTypeSystem, "You are a helpful assistant."))
	//messages = append(messages, llms.TextParts(llms.ChatMessageTypeHuman, "当前运行着多少设备"))
	//res, err := llm.GenerateContent(ctx, messages)
	//helper.Logger.Info("", zap.Any("res", res))
	return nil

}
