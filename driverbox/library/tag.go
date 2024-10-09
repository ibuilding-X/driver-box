package library

import (
	"encoding/json"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"os"
	"path"
)

const (
	tagDir  = "tag"
	tagFile = "tag.json"
)

var cacheTags []Tag

type Tag struct {
	// Key 唯一标识
	Key string `json:"key"`
	// Desc 标签描述（国际化）
	Desc map[string]string `json:"desc"`
}

func (t Tag) GetDesc(lang ...string) string {
	if len(lang) > 0 {
		return t.Desc[lang[0]]
	}

	return t.Desc[defaultLanguage]
}

// LoadTags 加载所有标签（缓存）
func LoadTags() []Tag {
	if cacheTags == nil {
		filePath := path.Join(config.ResourcePath, baseDir, tagDir, tagFile)
		if bs, err := os.ReadFile(filePath); err == nil {
			_ = json.Unmarshal(bs, &cacheTags)
		}
	}

	return cacheTags
}
