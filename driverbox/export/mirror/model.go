package mirror

// 自动生成镜像的配置结构
type autoMirrorConfig struct {
	ModelId     string `json:"modelId"`
	Description string `json:"description"`
	DriverKey   string `json:"driverKey"`
	Points      []map[string]string
}
