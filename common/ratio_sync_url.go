package common

// ValidateRatioSyncURL 与 ValidateOutboundURL 一致，供倍率同步等管理端拉取使用。
func ValidateRatioSyncURL(urlStr string) error {
	return ValidateOutboundURL(urlStr)
}
