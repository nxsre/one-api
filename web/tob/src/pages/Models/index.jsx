import PagePlaceholder from '@/components/PagePlaceholder';

export default function ModelsPage() {
  return (
    <PagePlaceholder
      title="模型广场"
      description="展示可用模型与渠道信息，面向 toB 客户的模型选型入口。"
      apiNote="复用 one-api 渠道列表 API（/api/channel 等）"
      defaultPath="/channel"
    />
  );
}
