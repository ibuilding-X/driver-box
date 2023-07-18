package edgex

import "github.com/ibuilding-x/driver-box/driverbox/config"

type ServiceConfig struct {
	DriverConfig config.DriverConfig
}

// UpdateFromRaw updates the service's full configuration from raw data received from
// the Service Provider.
func (sw *ServiceConfig) UpdateFromRaw(rawConfig interface{}) bool {
	configuration, ok := rawConfig.(*ServiceConfig)
	if !ok {
		return false //errors.New("unable to cast raw config to type 'ServiceConfig'")
	}
	*sw = *configuration
	return true
}
