package config

type ServiceConfig struct {
	DriverConfig DriverConfig
}

type DriverConfig struct {
	LoggerLevel      string // 日志等级
	PointCacheTTL    int64  // 点位缓存默认过期时间，单位：秒
	DefaultDeviceTTL int64  // 默认设备影子生命周期
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

// Validate ensures your custom configuration has proper values.
// Example of validating the sample custom configuration
func (scc *DriverConfig) Validate() error {
	return nil
}
