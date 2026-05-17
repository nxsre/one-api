package model

// 渠道编辑扩展分类（前端展示）；类型下拉完全由 relay/channeltype.BuiltinEditorTypes 提供，不再使用 channel_editor_types 表。
const (
	ChannelEditorTypeKindBasic  = "basic"
	ChannelEditorTypeKindCustom = "custom"
)
