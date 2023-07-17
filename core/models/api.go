package models

import (
	"github.com/ibuilding-x/driver-box/core/config"
)

// APIConfig restful API request body
type APIConfig struct {
	Key    string        `json:"key"`
	Config config.Config `json:"config"`
	Script string        `json:"script"`
}
