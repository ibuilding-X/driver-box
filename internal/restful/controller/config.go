package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/common"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/models"
	"github.com/ibuilding-x/driver-box/internal/bootstrap"
	"go.uber.org/zap"
	"io"
	"net/http"
	"os"
	"path"
)

type Config struct {
}

func (c *Config) Update(r *http.Request) (any, error) {
	// ------------------------------------------------------------
	// 配置文件覆盖更新
	// ------------------------------------------------------------
	// 接收核心配置
	body, err := io.ReadAll(r.Body)
	if err != nil {
		helper.Logger.Error("read body error", zap.Error(err))
		return nil, fmt.Errorf("read body error: %s", err)
	}
	defer r.Body.Close()

	// 数据解析
	var list []models.APIConfig
	err = json.Unmarshal(body, &list)
	if err != nil {
		helper.Logger.Error("config json decode error", zap.Error(err))
		return nil, fmt.Errorf("config json decode error: %s", err)
	}
	if len(list) == 0 {
		helper.Logger.Error("request body is empty")
		return nil, errors.New("request body is empty")
	}

	// 保存核心配置
	for _, config := range list {
		dir := path.Join(helper.EnvConfig.ConfigPath, config.Key)
		// 删除旧配置
		_ = os.RemoveAll(dir)
		// 创建文件夹
		_ = os.Mkdir(dir, 0755)
		// 保存 config.json / converter.lua 文件
		configFileName := path.Join(dir, common.CoreConfigName)
		scriptFilename := path.Join(dir, common.LuaScriptName)
		configData, _ := json.MarshalIndent(config.Config, "", "\t")

		err = os.WriteFile(configFileName, configData, 0666)
		if err != nil {
			helper.Logger.Error("save config.json file error", zap.Error(err))
		}
		if config.Config.ProtocolName == "modbus" && config.Script == "" {
			continue
		}
		err = os.WriteFile(scriptFilename, []byte(config.Script), 0666)
		if err != nil {
			helper.Logger.Error("save converter.lua file error", zap.Error(err))
		}
	}

	// ------------------------------------------------------------
	// plugins 重载
	// ------------------------------------------------------------
	// 1. 停止所有 timerTask 任务
	helper.Crontab.Stop()

	// 2. 停止运行中的 plugin
	pluginKeys := helper.CoreCache.GetAllRunningPluginKey()
	if len(pluginKeys) > 0 {
		for i, _ := range pluginKeys {
			if plugin, ok := helper.CoreCache.GetRunningPluginByKey(pluginKeys[i]); ok {
				err = plugin.Destroy()
				if err != nil {
					helper.Logger.Error("stop plugin error", zap.String("plugin", pluginKeys[i]), zap.Error(err))
				} else {
					helper.Logger.Info("stop plugin success", zap.String("plugin", pluginKeys[i]))
				}
			}
		}
	}
	// 3. 停止影子服务设备状态监听、删除影子服务
	helper.DeviceShadow.StopStatusListener()
	helper.DeviceShadow = nil

	// 4. 加载 plugins
	err = bootstrap.LoadPlugins()
	if err != nil {
		return nil, fmt.Errorf("load plugins error: %s", err)
	}

	return nil, nil
}
