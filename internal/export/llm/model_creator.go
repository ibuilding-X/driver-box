package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/library"
	"github.com/ibuilding-x/driver-box/internal/plugins"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/schema"
	"go.uber.org/zap"
)

type Response struct {
	ModelKey string `json:"modelKey"`
	SlaveId  string `json:"slaveId"`
	BusId    string `json:"busId"`
}

func (export *Export) AddDevice(deviceInfo string) {
	ctx := context.Background()
	//load data from PDF to add context on prompt
	//docsFromPdf, err := fetchDocumentsFromPdf(ctx)
	//if err != nil {
	//	log.Fatalln(err)
	//}
	//log.Default().Printf("Documents find on the PDF : %d", len(docsFromPdf))

	models := make([]config.DeviceModel, 0)
	docs := make([]schema.Document, 0)
	for _, modelKey := range library.Model().ListModels() {
		model, _ := library.Model().LoadLibrary(modelKey)
		models = append(models, model)
		//b, _ := json.Marshal(model)
		docs = append(docs, schema.Document{PageContent: "modelKey: " + modelKey + " 的物模型描述为:" + model.Description})
	}

	chain := chains.LoadStuffQA(export.llm)
	answer, _ := chains.Call(ctx, chain, map[string]any{
		"input_documents": docs,
		"question":        deviceInfo + " 适合用哪些物模型,以json格式告诉我这个模型对应的modelKey,从机地址(slaveId)和串口号(busId)。modelKey、slaveId、busId为同级字段,且都为字符串类型",
	})
	resp := Response{}
	a := answer["text"]
	b := fmt.Sprintf("%s", a)
	e := json.Unmarshal([]byte(b), &resp)
	fmt.Println(e)
	model, _ := library.Model().LoadLibrary(resp.ModelKey)
	model.Name = "aaa"
	e = helper.CoreCache.AddModel("modbus", model)
	helper.Logger.Info("response", zap.Any("resp", answer), zap.Error(e))

}

// 根据设备点表文件自动生成物模型
func (export *Export) CreateModel(file string) error {
	ctx := context.Background()
	//读取设备文档
	task := PdfReader{
		File: file,
	}
	e := task.Execute(ctx)
	if e != nil {
		return e
	}

	//解析设备采用的通信方式
	qa := DocumentQa{
		InputLlm:          export.llm,
		InputLlmDocuments: task.Result,
		InputQuestion:     "将该设备的通信方式以 protocol字段返回。如果识别出是modbus的rs485,则返回modbus",
	}
	if e = qa.Execute(ctx); e != nil {
		return e
	}
	protocol := qa.GetResult("protocol")

	plugin := export.guessPlugin(qa, protocol, ctx)
	helper.Logger.Info("guess result", zap.Any("plugin", plugin))

	qa.InputLlmDocuments = task.Result
	export.guessModelPoint(qa, ctx)
	return nil
}

func (export *Export) guessPlugin(qa DocumentQa, protocol string, ctx context.Context) string {
	protocols := make([]schema.Document, 0)
	for _, p := range plugins.Manager.GetSupportPlugins() {
		protocols = append(protocols, schema.Document{PageContent: "support protocol plugin : " + p})
	}
	helper.Logger.Info("support plugins", zap.Any("plugins", protocols))
	qa.InputLlmDocuments = protocols
	qa.InputQuestion = protocol + " match which protocol plugin, matched result key is: plugin"
	for i := 0; i < 3; i++ {
		qa.Execute(ctx)
		plugin := qa.GetResult("plugin")
		if plugin != "" {
			helper.Logger.Info("response", zap.Any("resp", plugin))
			return plugin
		} else {
			helper.Logger.Info("response", zap.Any("resp", "not found"))
		}
	}
	return ""
}

func (export *Export) guessModelPoint(qa DocumentQa, ctx context.Context) string {
	qa.InputQuestion = "这个设备的有效从机地址是多少"
	for i := 0; i < 1; i++ {
		qa.Execute(ctx)
		points := qa.GetResult("points")
		if points != "" {
			helper.Logger.Info("response", zap.Any("resp", points))
			return points
		} else {
			helper.Logger.Info("response", zap.Any("resp", "not found"))
		}
	}
	return ""
}
