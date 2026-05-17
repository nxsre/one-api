package model

// CatalogProviderAggRow 按 provider_key 聚合（仅用于管理端展示，字段已最小化）
type CatalogProviderAggRow struct {
	ProviderKey     string `gorm:"column:provider_key"`
	ProviderDisplay string `gorm:"column:provider_display"`
	NpmPackage      string `gorm:"column:npm_package"`
	Cnt             int64  `gorm:"column:cnt"`
}

// ListModelCatalogProviderAggregates 按提供方聚合启用中的目录行（需已同步 provider_key / npm_package）
func ListModelCatalogProviderAggregates() ([]CatalogProviderAggRow, error) {
	var rows []CatalogProviderAggRow
	err := DB.Model(&ModelCatalog{}).
		Select("provider_key, MAX(provider_display) AS provider_display, MAX(npm_package) AS npm_package, COUNT(*) AS cnt").
		Where("enabled = ? AND provider_key != ?", true, "").
		Group("provider_key").
		Order("provider_key").
		Scan(&rows).Error
	return rows, err
}
