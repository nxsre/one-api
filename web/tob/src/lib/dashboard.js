const QUOTA_DIVISOR = 1_000_000;

export const CHART_COLORS = {
  requests: '#6366f1',
  quota: '#06b6d4',
  tokens: '#8b5cf6',
};

export const BAR_COLORS = [
  '#6366f1',
  '#06b6d4',
  '#8b5cf6',
  '#10b981',
  '#f59e0b',
  '#ec4899',
  '#14b8a6',
  '#818cf8',
  '#f97316',
  '#38bdf8',
];

export function todayLocalDateStr() {
  const d = new Date();
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${y}-${m}-${day}`;
}

export function formatChartDate(dateStr) {
  const date = new Date(dateStr);
  return date.toLocaleDateString('zh-CN', { month: 'numeric', day: 'numeric' });
}

export function calculateSummary(dashboardData) {
  if (!Array.isArray(dashboardData) || dashboardData.length === 0) {
    return { todayRequests: 0, todayQuota: 0, todayTokens: 0 };
  }

  const today = todayLocalDateStr();
  const todayData = dashboardData.filter((item) => item.Day === today);

  return {
    todayRequests: todayData.reduce((sum, item) => sum + (item.RequestCount || 0), 0),
    todayQuota: todayData.reduce((sum, item) => sum + (item.Quota || 0), 0) / QUOTA_DIVISOR,
    todayTokens: todayData.reduce(
      (sum, item) => sum + (item.PromptTokens || 0) + (item.CompletionTokens || 0),
      0
    ),
  };
}

function getDateRangeBounds(dates) {
  let maxDate = new Date();
  let minDate =
    dates.length > 0 ? new Date(Math.min(...dates.map((d) => new Date(d)))) : new Date();

  const dataMaxDate =
    dates.length > 0 ? new Date(Math.max(...dates.map((d) => new Date(d)))) : new Date();
  if (dataMaxDate > maxDate) maxDate = dataMaxDate;

  const sevenDaysAgo = new Date();
  sevenDaysAgo.setDate(sevenDaysAgo.getDate() - 6);
  if (minDate > sevenDaysAgo) minDate = sevenDaysAgo;

  return { minDate, maxDate };
}

function fillDateRange(dailyMap, minDate, maxDate, fillEmpty) {
  for (let d = new Date(minDate); d <= maxDate; d.setDate(d.getDate() + 1)) {
    const y = d.getFullYear();
    const m = String(d.getMonth() + 1).padStart(2, '0');
    const day = String(d.getDate()).padStart(2, '0');
    const dateStr = `${y}-${m}-${day}`;
    if (!dailyMap[dateStr]) dailyMap[dateStr] = fillEmpty(dateStr);
  }
}

/** 按日聚合：请求 / 额度(M) / Token */
export function processTimeSeriesData(data) {
  const dailyData = {};

  data.forEach((item) => {
    if (!dailyData[item.Day]) {
      dailyData[item.Day] = { date: item.Day, requests: 0, quota: 0, tokens: 0 };
    }
    dailyData[item.Day].requests += item.RequestCount || 0;
    dailyData[item.Day].quota += (item.Quota || 0) / QUOTA_DIVISOR;
    dailyData[item.Day].tokens += (item.PromptTokens || 0) + (item.CompletionTokens || 0);
  });

  const { minDate, maxDate } = getDateRangeBounds(Object.keys(dailyData));
  fillDateRange(dailyData, minDate, maxDate, (dateStr) => ({
    date: dateStr,
    requests: 0,
    quota: 0,
    tokens: 0,
  }));

  return Object.values(dailyData).sort((a, b) => a.date.localeCompare(b.date));
}

/** 按日 × 模型堆叠 Token */
export function processModelData(data) {
  const models = [
    ...new Set(data.map((item) => item.ModelName).filter(Boolean)),
  ];
  const timeData = {};

  data.forEach((item) => {
    if (!item.ModelName) return;
    if (!timeData[item.Day]) {
      timeData[item.Day] = { date: item.Day };
      models.forEach((model) => {
        timeData[item.Day][model] = 0;
      });
    }
    timeData[item.Day][item.ModelName] += (item.PromptTokens || 0) + (item.CompletionTokens || 0);
  });

  const { minDate, maxDate } = getDateRangeBounds(Object.keys(timeData));
  fillDateRange(timeData, minDate, maxDate, (dateStr) => {
    const row = { date: dateStr };
    models.forEach((model) => {
      row[model] = 0;
    });
    return row;
  });

  return { series: Object.values(timeData).sort((a, b) => a.date.localeCompare(b.date)), models };
}

export function getBarColor(index) {
  return BAR_COLORS[index % BAR_COLORS.length];
}
