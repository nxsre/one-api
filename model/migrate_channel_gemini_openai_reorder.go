package model

import (
	"errors"
	"fmt"
	"strings"

	"github.com/songquanpeng/one-api/common/logger"
	"gorm.io/gorm"
)

const channelEnumGeminiOpenAIReorderKey = "ChannelEnumGeminiOpenAIReorderApplied"

func channelTypeGeminiOpenAIRemap(old int) int {
	if old == 46 {
		return 15
	}
	return old + 1
}

func buildChannelTypeGeminiOpenAIRemapCase(column string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "CASE %s ", column)
	for old := 46; old >= 15; old-- {
		fmt.Fprintf(&b, "WHEN %d THEN %d ", old, channelTypeGeminiOpenAIRemap(old))
	}
	fmt.Fprintf(&b, "ELSE %s END", column)
	return b.String()
}

// migrateChannelGeminiOpenAIReorderIfNeeded 应用紧凑枚举调整：GeminiOpenAICompatible 紧挨 Gemini，15..46 整体移位（46→15，其余 +1）；仅更新 channels.type。
func migrateChannelGeminiOpenAIReorderIfNeeded() error {
	var opt Option
	err := DB.Where("`key` = ?", channelEnumGeminiOpenAIReorderKey).First(&opt).Error
	if err == nil && strings.TrimSpace(opt.Value) == "true" {
		return nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	chCase := buildChannelTypeGeminiOpenAIRemapCase("type")

	return DB.Transaction(func(tx *gorm.DB) error {
		q1 := fmt.Sprintf("UPDATE channels SET type = %s WHERE type BETWEEN 15 AND 46", chCase)
		if err := tx.Exec(q1).Error; err != nil {
			return fmt.Errorf("migrate channels.type gemini openai reorder: %w", err)
		}
		rec := Option{Key: channelEnumGeminiOpenAIReorderKey, Value: "true"}
		if err := tx.Save(&rec).Error; err != nil {
			return err
		}
		logger.SysLog("channels: applied GeminiOpenAICompatible reorder (" + channelEnumGeminiOpenAIReorderKey + ")")
		return nil
	})
}
