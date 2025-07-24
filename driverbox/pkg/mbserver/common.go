package mbserver

const (
	ValueTypeBool = iota
	ValueTypeUint8
	ValueTypeInt8
	ValueTypeUint16
	ValueTypeInt16
	ValueTypeUint32
	ValueTypeInt32
	ValueTypeUint64
	ValueTypeInt64
	ValueTypeUint
	ValueTypeInt
	ValueTypeFloat32
	ValueTypeFloat64
)

const (
	AccessRead = iota
	AccessWrite
	AccessReadWrite
)

type Model struct {
	Id         string     `json:"id"`
	Name       string     `json:"name"` // 可选
	Properties []Property `json:"properties"`
}

type Property struct {
	Name        string `json:"name"`
	Description string `json:"description"` // 可选
	ValueType   int    `json:"valueType"`
	Access      int    `json:"access"`
}

type Device struct {
	ModelId string `json:"modelId"`
	Id      string `json:"id"`
}

type DeviceUnit struct {
	Id         string         `json:"id"`
	ModelId    string         `json:"modelId"`
	ModelName  string         `json:"modelName"`
	Properties []PropertyUnit `json:"properties"`
}

type PropertyUnit struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	Type         string `json:"type"`
	Access       string `json:"access"`
	StartAddress uint16 `json:"address"`
	Quantity     uint16 `json:"quantity"`
	HumanAddress string `json:"humanAddress"`
}

func ValueTypeText(valueType int) string {
	switch valueType {
	case ValueTypeBool:
		return "bool"
	case ValueTypeUint8:
		return "uint8"
	case ValueTypeInt8:
		return "int8"
	case ValueTypeUint16:
		return "uint16"
	case ValueTypeInt16:
		return "int16"
	case ValueTypeUint32:
		return "uint32"
	case ValueTypeInt32:
		return "int32"
	case ValueTypeUint64:
		return "uint64"
	case ValueTypeInt64:
		return "int64"
	case ValueTypeUint:
		return "uint"
	case ValueTypeInt:
		return "int"
	case ValueTypeFloat32:
		return "float32"
	case ValueTypeFloat64:
		return "float64"
	default:
		return ""
	}
}

func AccessText(access int) string {
	switch access {
	case AccessRead:
		return "R"
	case AccessWrite:
		return "W"
	case AccessReadWrite:
		return "RW"
	default:
		return ""
	}
}
