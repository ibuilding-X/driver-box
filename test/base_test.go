package test

import (
	"github.com/ibuilding-x/driver-box/internal/logger"
	"github.com/ibuilding-x/driver-box/pkg/driverbox/config"
)

func Init() {
	config.ResourcePath = "../res"
	logger.InitLogger("", "debug")
}
