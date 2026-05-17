/** 管理端拉取 /api/model_catalog/editor_options 后写入，供 getChannelOption / 列表渲染（含 default_base_url） */
let channelTypeOptionsCache = null;
let channelMap = undefined;

export function setChannelTypeOptionsCache(opts) {
  channelTypeOptionsCache = Array.isArray(opts) ? opts : null;
  channelMap = undefined;
}

export function getChannelOption(channelId) {
  const src = channelTypeOptionsCache || [];
  if (channelMap === undefined) {
    channelMap = {};
    src.forEach((option) => {
      channelMap[option.key] = option;
      channelMap[option.value] = option;
    });
  }
  return channelMap[channelId];
}
