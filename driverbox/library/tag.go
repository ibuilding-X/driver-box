package library

import (
	"encoding/json"
	"os"
	"path"
	"sync"

	"github.com/ibuilding-x/driver-box/driverbox/pkg/config"
	"golang.org/x/exp/slices"
)

const (
	tagDir  = "tag"
	tagFile = "tag.json"
)

var UsageTag = &usageTag{
	language: "zh-CN",
}

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

	return t.Desc[UsageTag.language]
}

type usageTag struct {
	language  string
	cacheTags []Tag
	lock      sync.Mutex
}

func (ut *usageTag) SetLanguage(lang string) {
	if lang != "" {
		ut.language = lang
	}
}

func (ut *usageTag) All() []Tag {
	ut.lock.Lock()
	defer ut.lock.Unlock()

	if ut.cacheTags == nil {
		filePath := path.Join(config.ResourcePath, baseDir, tagDir, tagFile)
		if bs, err := os.ReadFile(filePath); err == nil {
			_ = json.Unmarshal(bs, &ut.cacheTags)
		}
	}

	if len(ut.cacheTags) == 0 {
		return nil
	}

	result := make([]Tag, len(ut.cacheTags))
	copy(result, ut.cacheTags)
	return result
}

func (ut *usageTag) Get(key string) (Tag, bool) {
	tags := ut.All()
	for _, tag := range tags {
		if tag.Key == key {
			return tag, true
		}
	}
	return Tag{}, false
}

func (ut *usageTag) Filter(filter []string) []Tag {
	tags := ut.All()
	if len(filter) == 0 {
		return tags
	}

	result := make([]Tag, 0, len(tags))
	for _, tag := range tags {
		if slices.Contains(filter, tag.Key) {
			result = append(result, tag)
		}
	}
	return result
}
