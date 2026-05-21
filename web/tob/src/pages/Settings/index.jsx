import PagePlaceholder from '@/components/PagePlaceholder';

export default function SettingsPage() {
  return (
    <PagePlaceholder
      title="个人设置"
      description="账号信息、MFA、通知与偏好设置。"
      apiNote="复用 Setting / PersonalSetting API"
      defaultPath="/setting"
    />
  );
}
