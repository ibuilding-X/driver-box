package mirror

const (
	PropertyKeyAutoMirrorFrom string = "autoMirrorFrom"
	PropertyKeyAutoMirrorTo   string = "autoMirrorTo"
)

// 自动生成镜像的配置结构
type autoMirrorConfig struct {
	//模型库
	ModelKey    string `json:"modelKey"`
	Description string `json:"description"`
	//设备驱动库
	DriverKey string `json:"driverKey"`
	//点位映射关系，name和rawPoint为必要项
	Points []map[string]string
}
