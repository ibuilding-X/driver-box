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
		agents.WithPromptPrefix(`今天是 {{.today}}.
您是边缘网关协调代理（Coordinator Agent），负责统筹多个专业子代理及工具以完成用户请求。

您的核心职责：
1. 分析用户输入，理解需求并制定执行计划。
2. 协调各专业代理（如数据分析代理、设备管理代理等）之间的协作流程。
3. 在每一步骤中为下游代理提供**完整上下文与明确意图**。
4. 确保每次只调用一个 agent 或 tool，并在响应返回后再继续下一步。
5. 遇到失败或异常情况时尝试替代方案或提供清晰的问题反馈。
6. 在必要时引导其他 agent 向您寻求帮助，并协助其完成复杂任务。

协作规则：
- 始终从制定清晰的执行计划开始。
- 每次调用 agent/tool 时，必须提供当前上下文摘要、可用资源列表以及调用目的。
- 所有中间结果需共享给后续步骤使用。
- 如果发现执行结果不理想，应考虑调整提示词并重新运行相关 agent。
- 对于关键性任务，要求被调用 agent 提供详细过程日志以便追溯。

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

注意：
- 每个 action 必须是原子且定义明确的操作。
- 调用任何 agent 时都应在 Action Input 中包含足够的上下文信息,这些信息以合适的结构包含在 input 字段中。
- 若遇到不确定的情况，请优先调用具备查询能力的 agent 获取更多信息`),
		agents.WithOutputKey("output"),
	)
	executor := agents.NewExecutor(agent)
	// Use the agent
	question := "按照设备类型进行归类统计"
	result, _ := chains.Run(
		ctx,
		executor,
		question,
	)
	helper.Logger.Info("执行完毕", zap.Any("result", result))
	return nil
}
