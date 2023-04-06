// 解析配置默认格式为：JSON

package config

import (
	"driver-box/driver/common"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// ParseFromString 从字符串解析
func ParseFromString(s string) (c Config, err error) {
	err = json.Unmarshal([]byte(s), &c)
	return
}

// ParseFromFile 从文件解析
func ParseFromFile(path string) (c Config, err error) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	body, err := io.ReadAll(f)
	if err != nil {
		return
	}
	return ParseFromString(string(body))
}

// ParseFromPath 从指定路径解析所有核心配置文件
// directoryName => Config, example: http_server_sp200 => Config
func ParseFromPath(path string) (list map[string]Config, err error) {
	// 获取所有子目录
	dirs := make([]string, 0)
	if err = filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if err != nil { // 修复遍历文件夹错误（遍历不存在目录时，d 参数会返回 nil）
			return err
		}
		if d.IsDir() {
			dirs = append(dirs, d.Name())
		}
		return nil
	}); err != nil {
		return
	}

	if len(dirs) == 0 {
		return nil, errors.New("not found core config from ./driver-config")
	}

	list = make(map[string]Config)
	for i, _ := range dirs {
		fileName := filepath.Join(common.CoreConfigPath, dirs[i], common.CoreConfigName)
		c, err := ParseFromFile(fileName)
		if err != nil {
			continue
		}
		c.Key = dirs[i] // 复写 config key
		list[dirs[i]] = c
	}
	return
}
