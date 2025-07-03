package agent

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
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
		agents.WithPromptPrefix(`Today is {{.today}}.
You are a device management agent running on an edge gateway.
Your role is to provide accurate device-related information and execute device operations as requested by the coordinator agent.

Key responsibilities:
1. Accurately identify devices based on natural language descriptions
2. Retrieve real-time device status and metrics
3. Execute control commands on devices
4. Monitor device health and report anomalies
5. Provide device-specific knowledge for decision-making

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
