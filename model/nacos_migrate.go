package model

// migrateNacosExtensionTables 创建 Nacos 控制台扩展、配置中心、AI 注册等表（启动时 AutoMigrate）。
func migrateNacosExtensionTables() error {
	if err := DB.AutoMigrate(
		&NacosCsConfig{},
		&NacosCsConfigHistory{},
		&NacosCsConfigBeta{},
		&NacosCsConfigListener{},
		&NacosConsoleDiscoveryService{},
		&NacosConsoleDiscoveryInstance{},
		&NacosConsoleSubscriber{},
		&NacosConsolePlugin{},
		&NacosConsoleClusterNode{},
		&NacosUserACL{},
		&NacosAIArtifact{},
		&NacosAIArtifactVersion{},
		&NacosAIMcpServer{},
		&NacosAIA2AAgent{},
		&NacosAIPrompt{},
		&NacosAIPromptVersion{},
		&NacosAIPipelineRun{},
	); err != nil {
		return err
	}
	return SeedNacosConsoleFacilityDefaults(DB)
}
