package model

import "github.com/songquanpeng/one-api/setting/billing_setting"

func init() {
	billing_setting.RegisterPricingVersionStoreSaver = UpdateOption
	billing_setting.RegisterPricingEntrySnapshot = SnapshotPricingEntriesToVersion
}
