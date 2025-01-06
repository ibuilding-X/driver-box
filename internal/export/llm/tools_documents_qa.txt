package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/schema"
)

type DocumentQa struct {
	InputLlm          llms.Model
	InputLlmDocuments []schema.Document
	InputQuestion     string
	answer            map[string]any
}

func (task *DocumentQa) Execute(context context.Context) error {
	chain := chains.LoadStuffQA(task.InputLlm)
	answer, _ := chains.Call(context, chain, map[string]any{
		"input_documents": task.InputLlmDocuments,
		"question":        task.InputQuestion,
	})
	fmt.Println(answer)
	task.answer = answer
	return nil
}

func (task *DocumentQa) GetResult(name string) string {
	a := task.answer["text"]
	b := fmt.Sprintf("%s", a)
	m := make(map[string]string)
	json.Unmarshal([]byte(b), &m)
	return m[name]
}
