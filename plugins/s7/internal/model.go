package internal

import (
	"time"

	"github.com/ibuilding-x/driver-box/v2/driverbox/plugin"
)

type ConnectionConfig struct {
	plugin.BaseConnection
	Address    string        `json:"address"`
	Rack       int           `json:"rack"`
	Slot       int           `json:"slot"`
	Interval   time.Duration `json:"interval"`
	Timeout    time.Duration `json:"timeout"`
	RetryCount int           `json:"retryCount"`
}

type NodeConfig struct {
	Area      string  `json:"area"`      // DB, M, I, Q
	DB        int     `json:"db"`        // DB number
	Start     int     `json:"start"`     // Start address
	Size      int     `json:"size"`      // Size in bytes
	DataType  string  `json:"dataType"`  // BOOL, BYTE, WORD, DWORD, INT, DINT, REAL
	Bit       int     `json:"bit"`       // Bit position (for BOOL)
	Scale     float64 `json:"scale"`     // Scale factor
	Writeable bool    `json:"writeable"` // Is writeable
	PointName string  `json:"pointName"` // Point name
}

type ReadRequest struct {
	Nodes []*NodeConfig
}

type WriteRequest struct {
	Node  *NodeConfig
	Value interface{}
}
