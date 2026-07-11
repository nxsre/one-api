/** toB 菜单 ↔ one-api default 业务路由对照 */
export const NAV_SECTIONS = [
  {
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
    ],
  },
  {
    label: '体验中心',
    items: [
      {
        key: 'playground',
        path: '/playground',
        label: '语言模型',
        apiNote: 'POST /v1/chat/completions',
        icon: 'playground',
      },
    ],
  },
  {
    label: '可观测性',
    items: [
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
    label: '管理',
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
