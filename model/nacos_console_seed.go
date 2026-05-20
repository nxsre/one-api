package model

import (
	"errors"

	"gorm.io/gorm"
)

// SeedNacosConsoleFacilityDefaults 控制台扩展表默认数据（幂等）。
func SeedNacosConsoleFacilityDefaults(db *gorm.DB) error {
	plugins := []NacosConsolePlugin{
		{PluginId: "com.alibaba.nacos.plugin.ai", PluginName: "AI Registry", PluginType: "AI", Enabled: true, Critical: false, Configurable: true, Exclusive: false, AvailableNodeCount: 1, TotalNodeCount: 1},
		{PluginId: "com.alibaba.nacos.plugin.auth", PluginName: "Auth", PluginType: "SECURITY", Enabled: true, Critical: true, Configurable: false, Exclusive: false, AvailableNodeCount: 1, TotalNodeCount: 1},
		{PluginId: "com.alibaba.nacos.plugin.config", PluginName: "Config", PluginType: "CONFIG", Enabled: true, Critical: false, Configurable: true, Exclusive: false, AvailableNodeCount: 1, TotalNodeCount: 1},
	}
	for i := range plugins {
		var x NacosConsolePlugin
		err := db.Where("plugin_id = ?", plugins[i].PluginId).First(&x).Error
		if err == nil {
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := db.Create(&plugins[i]).Error; err != nil {
			return err
		}
	}
	return nil
}
