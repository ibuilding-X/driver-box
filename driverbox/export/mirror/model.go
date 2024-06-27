package mirror

// 自动生成镜像的配置结构
type autoMirrorConfig struct {
	ModelId   string `json:"modelId"`
	DriverKey string `json:"driverKey"`
	Points    []map[string]string
}
