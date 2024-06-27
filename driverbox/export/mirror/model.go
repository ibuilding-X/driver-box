package mirror

type autoMirrorConfig struct {
	ModelId   string `json:"modelId"`
	DriverKey string `json:"driverKey"`
	Points    []map[string]string
}
