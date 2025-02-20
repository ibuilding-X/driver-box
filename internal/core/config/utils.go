package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/dto"
	"os"
	"strconv"
)

// 删除无用的模型、连接
func optimize(c dto.Config) dto.Config {
	config := clone(c)

	usefulConnKeys := make(map[string]struct{})
	for modelName, model := range c.Models {
		// 删除没有设备的模型
		if len(model.Devices) == 0 {
			delete(config.Models, modelName)
			continue
		}

		// 获取使用中的连接
		for _, device := range model.Devices {
			usefulConnKeys[device.ConnectionKey] = struct{}{}
		}
	}

	connections := make(map[string]dto.H)
	for key, _ := range usefulConnKeys {
		connections[key] = config.Connections[key]
	}
	config.Connections = connections

	return config
}

// 克隆配置
func clone(c dto.Config) dto.Config {
	bs, _ := json.Marshal(c)
	var config dto.Config
	_ = json.Unmarshal(bs, &config)
	return config
}

// map 转 point
func hToPoint(h dto.H) dto.Point {
	var point dto.Point
	point.Extends = make(dto.H)

	for k, v := range h {
		switch k {
		case "name":
			point.Name = fmt.Sprintf("%v", v)
		case "description":
			point.Description = fmt.Sprintf("%v", v)
		case "valueType":
			point.ValueType = dto.ValueType(fmt.Sprintf("%v", v))
		case "readWrite":
			point.ReadWrite = dto.ReadWrite(fmt.Sprintf("%v", v))
		case "reportMode":
			point.ReportMode = dto.ReportMode(fmt.Sprintf("%v", v))
		case "units":
			point.Units = fmt.Sprintf("%v", v)
		case "scale":
			switch v.(type) {
			case float64:
				point.Scale = v.(float64)
			case int64:
				point.Scale = float64(v.(int64))
			}
		case "decimals":
			if decimals, err := strconv.Atoi(fmt.Sprintf("%v", v)); err == nil {
				point.Decimals = decimals
			}
		case "enums":
			enums := make([]dto.PointEnum, 0)
			if b, err := json.Marshal(v); err == nil {
				_ = json.Unmarshal(b, &enums)
				point.Enums = enums
			}
		default:
			point.Extends[k] = v
		}
	}

	return point
}

// point 转 map
func pointToH(point dto.Point) dto.H {
	h := make(dto.H)
	h["name"] = point.Name
	h["description"] = point.Description
	h["valueType"] = string(point.ValueType)
	h["readWrite"] = string(point.ReadWrite)
	h["reportMode"] = string(point.ReportMode)
	h["units"] = point.Units
	h["scale"] = point.Scale
	h["decimals"] = point.Decimals
	h["enums"] = point.Enums

	// extensions
	for k, v := range point.Extends {
		h[k] = v
	}
	return h
}

// modelMetadata 转 model
func metadataToModel(protocolName string, metadata modelMetadata) dto.Model {
	model := dto.Model{
		Name:         metadata.Name,
		ModelId:      metadata.ModelId,
		Description:  metadata.Description,
		Attributes:   metadata.Attributes,
		ProtocolName: protocolName,
	}

	// points
	points := make(map[string]dto.Point)
	for _, point := range metadata.Points {
		if p := hToPoint(point); p.Name != "" {
			points[p.Name] = p
		}
	}
	model.Points = points

	// devices
	devices := make(map[string]dto.Device)
	for _, device := range metadata.Devices {
		// 填充冗余字段
		device.ProtocolName = protocolName
		device.ModelName = model.Name
		device.ModelId = model.ModelId

		devices[device.Id] = device
	}
	model.Devices = devices

	return model
}

// model 转 modelMetadata
func modelToMetadata(model dto.Model) modelMetadata {
	metadata := modelMetadata{
		Name:        model.Name,
		ModelId:     model.ModelId,
		Description: model.Description,
		Attributes:  model.Attributes,
	}

	// points
	points := make([]dto.H, 0, len(model.Points))
	for _, point := range model.Points {
		points = append(points, pointToH(point))
	}
	metadata.Points = points

	// devices
	devices := make([]dto.Device, 0, len(model.Devices))
	for _, device := range model.Devices {
		devices = append(devices, device)
	}
	metadata.Devices = devices

	return metadata
}

// config 转 configMetadata
func configToMetadata(config dto.Config) configMetadata {
	metadata := configMetadata{
		Connections:  config.Connections,
		ProtocolName: config.ProtocolName,
	}

	var models []modelMetadata
	for _, model := range config.Models {
		models = append(models, modelToMetadata(model))
	}
	metadata.Models = models

	return metadata
}

// 读取配置文件
func readFile(path string) (config dto.Config, err error) {
	if _, err = os.Stat(path); err != nil {
		return
	}

	// 读取文件
	bs, err := os.ReadFile(path)
	if err != nil {
		return
	}
	if len(bs) == 0 {
		err = errors.New("config file is empty")
		return
	}

	// 解析
	var metadata configMetadata
	err = json.Unmarshal(bs, &metadata)
	if err != nil {
		return
	}
	config = metadata.ToConfig()
	return
}

// 补全配置（填充冗余字段：模型、设备）
func complete(config dto.Config) dto.Config {
	for i, model := range config.Models {
		model.ProtocolName = config.ProtocolName
		for j, device := range model.Devices {
			device.ProtocolName = config.ProtocolName
			device.ModelName = model.Name
			device.ModelId = model.ModelId
			model.Devices[j] = device
		}
		config.Models[i] = model
	}
	return config
}
