package modbus

type ValueType uint8 // 值类型

type Access uint8 // 访问权限

const (
	ValueTypeUint16 ValueType = iota
	ValueTypeFloat32
)

const (
	AccessRead Access = iota
	AccessWrite
	AccessReadWrite
)

type Model struct {
	Id         string     `json:"id"`
	Properties []Property `json:"properties"`

	need int
}

type Property struct {
	Name      string    `json:"name"`
	ValueType ValueType `json:"valueType"`
	Access    Access    `json:"access"`
}

type Device struct {
	Mid string `json:"mid"`
	Id  string `json:"id"`
}

func (v ValueType) String() string {
	switch v {
	case ValueTypeUint16:
		return "uint16"
	case ValueTypeFloat32:
		return "float32"
	default:
		return "unknown"
	}
}
