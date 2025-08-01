package mcp

import (
	"github.com/ibuilding-x/driver-box/driverbox"
	"github.com/ibuilding-x/driver-box/internal/export/ai"
)

func LoadMcpExport() {
	driverbox.Exports.LoadExport(ai.NewExport())
}
