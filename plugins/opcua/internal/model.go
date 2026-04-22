package internal

import (
	"time"

	"github.com/ibuilding-x/driver-box/v2/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/v2/pkg/config"
)

type ConnectionConfig struct {
	plugin.BaseConnection
	Endpoint   string        `json:"endpoint"`
	Username   string        `json:"username"`
	Password   string        `json:"password"`
	Policy     string        `json:"policy"`
	Mode       string        `json:"mode"`
	CertFile   string        `json:"certFile"`
	KeyFile    string        `json:"keyFile"`
	Interval   time.Duration `json:"interval"`
	Timeout    time.Duration `json:"timeout"`
	RetryCount int           `json:"retryCount"`
}

type NodeConfig struct {
	NodeId      string      `json:"nodeId"`
	Namespace   int         `json:"namespace"`
	PointName   string      `json:"pointName"`
	DataType    string      `json:"dataType"`
	Writeable   bool        `json:"writeable"`
	Scale       float64     `json:"scale"`
	Description string      `json:"description"`
}

type OpcuaPoint struct {
	config.Point
	NodeConfig
}

type ReadRequest struct {
	Nodes []string
}

type WriteRequest struct {
	NodeId string
	Value  interface{}
}
