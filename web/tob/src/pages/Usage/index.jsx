import PagePlaceholder from '@/components/PagePlaceholder';

export default function UsagePage() {
  return (
    <PagePlaceholder
      title="用量统计"
      description="平台财务报表、用量趋势与计费概览。"
      apiNote="复用 PlatformReports 相关 API"
      defaultPath="/platform-reports"
    />
  );
}
