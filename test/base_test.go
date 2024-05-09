package test

import (
	"github.com/ibuilding-x/driver-box/internal/library"
	"github.com/ibuilding-x/driver-box/internal/logger"
	"path"
)

func Init() {
	library.BaseDir = path.Join("../res", "library")
	logger.InitLogger("", "debug")
}
