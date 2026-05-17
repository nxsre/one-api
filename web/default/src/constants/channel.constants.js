/**
 * 渠道类型选项已改为运行时从 GET /api/model_catalog/editor_options 加载（后端 BuiltinEditorTypes + default_base_urls）。
 * 保留空数组仅兼容旧 import；请勿再在此处扩充类型。
 */
export const CHANNEL_OPTIONS = [];
