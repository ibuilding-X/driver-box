package mcp

import (
	"github.com/ibuilding-x/driver-box/internal/export/ai"
	"github.com/ibuilding-x/driver-box/pkg/driverbox"
)

func LoadMcpExport() {
	driverbox.Exports.LoadExport(ai.NewExport())
}
