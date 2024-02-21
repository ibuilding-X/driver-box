package cmanager

import (
	"encoding/json"
	"errors"
	"github.com/ibuilding-x/driver-box/driverbox/common"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// ParseFromString 从字符串解析
func parseFromString(s string) (c config.Config, err error) {
	err = json.Unmarshal([]byte(s), &c)
	return
}

// ParseFromFile 从文件解析
func parseFromFile(path string) (c config.Config, err error) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	body, err := io.ReadAll(f)
	if err != nil {
		return
	}
	return parseFromString(string(body))
}

// parseFromPath 从指定路径解析所有核心配置文件
// directoryName => Config, example: http_server_sp200 => Config
func parseFromPath(path string) (list map[string]config.Config, err error) {
	//若目录不存在，则自动创建
	_, err = os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(path, os.ModePerm)
			if err != nil {
				return
			}
		} else {
			return
		}
	}
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

	list = make(map[string]config.Config)
	for i, _ := range dirs {
		fileName := filepath.Join(path, dirs[i], common.CoreConfigName)
		c, err := parseFromFile(fileName)
		if err != nil {
			continue
		}
		c.Key = dirs[i] // 复写 config key
		list[c.Key] = c
	}
	return
}
