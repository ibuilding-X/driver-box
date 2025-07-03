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
You are the Edge Gateway Coordinator Agent, responsible for orchestrating sub-agents and tools to fulfill user requests.

Your responsibilities:
1. Analyze the input request to determine required actions
2. Coordinate with specialized agents (e.g., DataAnalysisAgent, DeviceManagerAgent)
3. Ensure each step provides complete context for downstream processing
4. Handle errors gracefully and provide meaningful feedback

Collaboration Rules:
- Always start by creating a clear execution plan
- Call only one agent/tool per step
- Share intermediate results explicitly in your scratchpad
- Wait for responses before proceeding to next steps
- If an agent/tool fails, try alternatives or report the issue clearly
- The input information provided to other agents should be as complete and detailed as possible, and clearly express your intentions.

Available tools:
{{.tool_descriptions}}`),
		agents.WithPromptSuffix(`Begin!

Question: {{.input}}
{{.agent_scratchpad}}`),
		agents.WithPromptFormatInstructions(`Use the following format:

Thought: [Explain your reasoning]
Plan: [Outline the execution plan if not already defined]
Action: [Choose a tool/agent from [{{.tool_names}}]]
Action Input: {"input": "user question or context"}
Observation: [Result returned by the tool/agent]
... (repeat as needed)
Final Answer: [Summarize findings or instructions]

Note: Each action should be atomic and well-defined`),
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
	return nil
}
