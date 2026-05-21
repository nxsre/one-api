/** toB 菜单 ↔ one-api default 业务路由对照 */
export const NAV_SECTIONS = [
  {
    label: '主导航',
    items: [
      {
        key: 'overview',
        path: '/overview',
        label: '概览',
        apiNote: 'GET /api/user/dashboard',
        icon: 'overview',
      },
      {
        key: 'models',
        path: '/models',
        label: '模型广场',
        apiNote: '渠道 /channel',
        icon: 'models',
      },
      {
        key: 'usage',
        path: '/usage',
        label: '用量统计',
        apiNote: '平台财务报表 /platform-reports',
        icon: 'usage',
      },
      {
        key: 'logs',
        path: '/logs',
        label: '日志',
        apiNote: '操作日志 /operation',
        icon: 'logs',
      },
    ],
  },
  {
    label: '账户',
    items: [
      {
        key: 'api-keys',
        path: '/api-keys',
        label: 'API KEY',
        apiNote: '令牌 /token',
        icon: 'key',
      },
      {
        key: 'settings',
        path: '/settings',
        label: '个人设置',
        apiNote: '系统设置 /setting',
        icon: 'settings',
      },
    ],
  },
];

export const ROUTE_TITLES = Object.fromEntries(
  NAV_SECTIONS.flatMap((s) => s.items.map((i) => [i.path, i.label]))
);
