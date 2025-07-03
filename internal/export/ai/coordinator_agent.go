package ai

import (
	"context"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	agent2 "github.com/ibuilding-x/driver-box/internal/export/ai/agent"
	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/tools"
	"go.uber.org/zap"
)

func (export *Export) startAgent() error {
	mcpTools, e := export.getLangChainTools()

	if e != nil {
		return e
	}

	ctx := context.Background()

	dataAnalysisAgent := &agent2.DataAnalysisAgent{
		LLM:   export.llm,
		Tools: mcpTools,
	}
	deviceManagerAgent := &agent2.DeviceManagerAgent{
		LLM:   export.llm,
		Tools: mcpTools,
	}

	tool := make([]tools.Tool, 0)
	//tool = append(tool, mcpTools...)
	tool = append(tool, dataAnalysisAgent)
	tool = append(tool, deviceManagerAgent)

	agent := agents.NewOneShotAgent(
		export.llm,
		tool,
		agents.WithMaxIterations(3),
		agents.WithPromptPrefix(`Today is {{.today}}.
You are a coordinator agent running on an edge gateway.
Your responsibility is to design a complete execution plan and use available tools and agents to meet user requests.

Collaboration Guidelines:
1. First, formulate an execution plan and define your role in the entire workflow.
2. Coordinate with other agents by:
   - Following the planned execution order
   - Sharing intermediate results through the shared context
   - Adjusting parameters based on feedback from previous steps
   - Reporting status updates for progress tracking
3. When executing your part of the task, focus on:
   - Maintaining compatibility with subsequent steps
   - Providing clear, structured outputs for downstream processing
   - Handling errors gracefully and providing meaningful error messages
   - The input information provided to other agents should be as complete and detailed as possible, and clearly express your intentions.
4. If conditions change during execution:
   - Assess impact on the overall plan
   - Coordinate with other agents to adjust timelines or parameters
   - Document any deviations from the original plan

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
	question := "对设备进行分类"
	result, _ := chains.Run(
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
	return nil
}
