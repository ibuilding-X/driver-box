package agent

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/ibuilding-x/driver-box/pkg/driverbox/helper"
	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/tools"
	"go.uber.org/zap"
)

type DeviceManagerAgent struct {
	LLM   llms.Model
	Tools []tools.Tool
}

func (t *DeviceManagerAgent) Name() string {
	return "device_manager_agent"
}

// Description returns the description of the tool along with its input schema.
func (t *DeviceManagerAgent) Description() string {
	return "Manages IoT devices including status queries, configurations, and control commands\nargs: {\"input\": \"string\"}"
}

// Call invokes the MCP tool with the given input and returns the result.
func (t *DeviceManagerAgent) Call(ctx context.Context, input string) (string, error) {
	c := make(map[string]string)
	e := json.Unmarshal([]byte(input), &c)
	if e != nil {
		return "", e
	}
	content, ok := c["input"]
	if !ok || content == "" {
		return "", errors.New("input is empty")
	}
	agent := agents.NewOneShotAgent(
		t.LLM,
		t.Tools,
		agents.WithMaxIterations(3),
		agents.WithPromptPrefix(`今天是 {{.today}}.
你是运行在边缘网关上的负责设备管理的 AI Agent。
你的职责是应 coordinator Agent 的请求，提供准确的设备相关信息并执行设备操作。

主要职责：
1. 根据自然语言描述准确识别设备
2. 获取设备实时状态和指标
4. 监测设备健康状况并报告异常情况
5. 为决策提供设备特定知识

Available tools:
{{.tool_descriptions}}`),
		agents.WithPromptSuffix(`Begin!

Question: {{.input}}
{{.agent_scratchpad}}`),
		agents.WithPromptFormatInstructions(`Use the following format:

Question: the input question you must answer
Thought: you should always think about what to do
Action: the action to take, should be one of [ {{.tool_names}} ]
Action Input: the input to the action
Observation: the result of the action
... (this Thought/Action/Action Input/Observation can repeat N times)
Thought: I now know the final answer
Final Answer: the final answer to the original input question`),
		agents.WithOutputKey("output"),
	)
	executor := agents.NewExecutor(agent)
	// Use the agent
	question := content
	result, err := chains.Run(
		ctx,
		executor,
		question,
	)
	helper.Logger.Info("执行完毕", zap.Any("result", result), zap.Error(err))
	if err != nil {
		return "", err
	}

	return t.Name() + " operations completed, Answer: \n" + result, nil
}
