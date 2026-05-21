import PagePlaceholder from '@/components/PagePlaceholder';

export default function ApiKeysPage() {
  return (
    <PagePlaceholder
      title="API KEY"
      description="令牌创建、轮换与权限管理。"
      apiNote="复用 Token 管理 API（/api/token）"
      defaultPath="/token"
    />
  );
}
