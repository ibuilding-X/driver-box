package main

import (
	"github.com/ibuilding-x/driver-box/driverbox"
	"github.com/ibuilding-x/driver-box/driverbox/export"
)

func main() {
	driverbox.Start([]export.Export{&export.DefaultExport{}})
}
