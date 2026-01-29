package agent

import (
	"context"
	"github.com/ibuilding-x/driver-box/v2/driverbox"
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
	return "Performs data analysis tasks such as classification, prediction, and pattern recognition on device data\nargs: {\"input\": \"string\"}"
}

// Call invokes the MCP tool with the given input and returns the result.
func (t *DataAnalysisAgent) Call(ctx context.Context, input string) (string, error) {

	agent := agents.NewOneShotAgent(
		t.LLM,
		t.Tools,
		agents.WithMaxIterations(3),
		agents.WithPromptPrefix(`今天是 {{.today}}.
你是一个运行在边缘网关上善于进行数据分析的 AI Agent, 。

您的职责：
1. 分析输入请求以确定所需操作。如果输入请求意图表达不清晰，需要给予反馈并提出你的要求。
2. 主要基于数据库相关 tool 处理请求，可适当采用其他 mcp tool 辅助
3. 确保每个步骤为下游处理提供完整上下文
4. 优雅处理错误并提供有效反馈

协作规则：
1. 始终从制定清晰执行计划开始
2. 每个步骤仅调用一个 tool
3. 收到响应后再继续后续步骤
4. 若 tool 执行失败，尝试替代方案或清晰报告问题
5. 若发现执行结果不符合预期，先尝试自检解决。若无法处理则明确返回给用户。

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
