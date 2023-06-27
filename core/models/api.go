package models

// APIConfig restful API request body
type APIConfig struct {
	Key    string      `json:"key"`
	Config interface{} `json:"config"`
	Script string      `json:"script"`
}
