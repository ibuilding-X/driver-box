package config

import (
	"github.com/ibuilding-x/driver-box/driverbox/dto"
)

type configMetadata struct {
	Models       []modelMetadata  `json:"deviceModels"`
	Connections  map[string]dto.H `json:"connections"`
	ProtocolName string           `json:"protocolName"`
}

type modelMetadata struct {
	Name        string       `json:"name"`
	ModelId     string       `json:"modelId"`
	Description string       `json:"description"`
	Attributes  dto.H        `json:"attributes"`
	Points      []dto.H      `json:"devicePoints"`
	Devices     []dto.Device `json:"devices"`
}

// ToConfig 配置文件转换
func (cm configMetadata) ToConfig() dto.Config {
	c := dto.Config{
		Connections:  cm.Connections,
		ProtocolName: cm.ProtocolName,
	}

	models := make(map[string]dto.Model)
	for _, model := range cm.Models {
		models[model.Name] = metadataToModel(cm.ProtocolName, model)
	}
	c.Models = models

	return c
}
