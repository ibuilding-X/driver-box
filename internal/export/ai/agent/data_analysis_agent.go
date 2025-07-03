package agent

import (
	"context"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/tools"
	"go.uber.org/zap"
)

type DataAnalysisAgent struct {
	LLM   llms.Model
	Tools []tools.Tool
}

func (t *DataAnalysisAgent) Name() string {
	return "data_analysis_agent"
}

// Description returns the description of the tool along with its input schema.
func (t *DataAnalysisAgent) Description() string {
	return "You are a data analyst for edge gateways."
}

// Call invokes the MCP tool with the given input and returns the result.
func (t *DataAnalysisAgent) Call(ctx context.Context, input string) (string, error) {

	agent := agents.NewOneShotAgent(
		t.LLM,
		t.Tools,
		agents.WithMaxIterations(3),
		agents.WithPromptPrefix(`Today is {{.today}}.
You are an intelligent agent running on an edge gateway.
Your role is to assist with device control, monitoring, and basic decision-making using the tools available.
You need to accurately and completely identify the set of devices pointed to by the user.

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
	question := input
	result, err := chains.Run(
		ctx,
		executor,
		question,
	)
	helper.Logger.Info("执行完毕", zap.Any("result", result))

	//messages := make([]llms.MessageContent, 0)
	//messages = append(messages, llms.TextParts(llms.ChatMessageTypeSystem, "You are a helpful assistant."))
	//messages = append(messages, llms.TextParts(llms.ChatMessageTypeHuman, "当前运行着多少设备"))
	//res, err := llm.GenerateContent(ctx, messages)
	//helper.Logger.Info("", zap.Any("res", res))
	return result, err
}
