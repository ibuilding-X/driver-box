package test

import (
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/internal/logger"
)

func Init() {
	config.ResourcePath = "../res"
	logger.InitLogger("", "debug")
}
