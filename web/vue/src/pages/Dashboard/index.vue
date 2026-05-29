<template>
  <div class="dashboard-container">
    <!-- 三个并排的折线图 -->
    <a-row :gutter="16" class="charts-grid">
      <a-col :xs="24" :md="8">
        <a-card class="chart-card">
          <div class="header">{{ t('dashboard.charts.requests.title') }}</div>
          <div class="chart-container">
            <v-chart class="echart" :option="requestsOption" :style="{ height: '120px' }" autoresize />
          </div>
        </a-card>
      </a-col>

      <a-col :xs="24" :md="8">
        <a-card class="chart-card">
          <div class="header">{{ t('dashboard.charts.quota.title') }}</div>
          <div class="chart-container">
            <v-chart class="echart" :option="quotaOption" :style="{ height: '120px' }" autoresize />
          </div>
        </a-card>
      </a-col>

      <a-col :xs="24" :md="8">
        <a-card class="chart-card">
          <div class="header">{{ t('dashboard.charts.tokens.title') }}</div>
          <div class="chart-container">
            <v-chart class="echart" :option="tokensOption" :style="{ height: '120px' }" autoresize />
          </div>
        </a-card>
      </a-col>
    </a-row>

    <!-- 模型使用统计 -->
    <a-card class="chart-card">
      <div class="header">{{ t('dashboard.statistics.title') }}</div>
      <div class="chart-container">
        <v-chart class="echart" :option="modelOption" :style="{ height: '300px' }" autoresize />
      </div>
    </a-card>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';
import axios from 'axios';
import { use } from 'echarts/core';
import { CanvasRenderer } from 'echarts/renderers';
import { LineChart, BarChart } from 'echarts/charts';
import {
  GridComponent,
  TooltipComponent,
  LegendComponent,
} from 'echarts/components';
import VChart from 'vue-echarts';

use([
  CanvasRenderer,
  LineChart,
  BarChart,
  GridComponent,
  TooltipComponent,
  LegendComponent,
]);

const { t } = useI18n();

const chartConfig = {
  colors: {
    requests: '#4318FF',
    quota: '#00B5D8',
    tokens: '#6C63FF',
  },
  barColors: [
    '#4318FF', // 深紫色
    '#00B5D8', // 青色
    '#6C63FF', // 紫色
    '#05CD99', // 绿色
    '#FFB547', // 橙色
    '#FF5E7D', // 粉色
    '#41B883', // 翠绿
    '#7983FF', // 淡紫
    '#FF8F6B', // 珊瑚色
    '#49BEFF', // 天蓝
  ],
};

const data = ref([]);

const fetchDashboardData = async () => {
  try {
    const response = await axios.get('/api/user/dashboard');
    if (response.data.success) {
      const dashboardData = response.data.data || [];
      data.value = dashboardData;
    }
  } catch (error) {
    console.error('Failed to fetch dashboard data:', error);
    data.value = [];
  }
};

// 处理数据以供折线图使用，补充缺失的日期
const processTimeSeriesData = () => {
  const dailyData = {};

  // 填充实际数据
  data.value.forEach((item) => {
    if (!dailyData[item.Day]) {
      dailyData[item.Day] = {
        date: item.Day,
        requests: 0,
        quota: 0,
        tokens: 0,
      };
    }
    dailyData[item.Day].requests += item.RequestCount;
    dailyData[item.Day].quota += item.Quota / 1000000;
    dailyData[item.Day].tokens += item.PromptTokens + item.CompletionTokens;
  });

  // 获取日期范围
  const dates = Object.keys(dailyData);
  let maxDate = new Date();
  let minDate =
    dates.length > 0
      ? new Date(Math.min(...dates.map((d) => new Date(d))))
      : new Date();

  const dataMaxDate =
    dates.length > 0
      ? new Date(Math.max(...dates.map((d) => new Date(d))))
      : new Date();
  if (dataMaxDate > maxDate) maxDate = dataMaxDate;

  // 确保至少显示7天的数据
  const sevenDaysAgo = new Date();
  sevenDaysAgo.setDate(sevenDaysAgo.getDate() - 6); // -6是因为包含今天
  if (minDate > sevenDaysAgo) {
    minDate = sevenDaysAgo;
  }

  // 生成所有日期，补全空缺
  for (let d = new Date(minDate); d <= maxDate; d.setDate(d.getDate() + 1)) {
    // 解决时区导致日期生成差异的问题，使用本地年月日
    const y = d.getFullYear();
    const m = String(d.getMonth() + 1).padStart(2, '0');
    const day = String(d.getDate()).padStart(2, '0');
    const dateStr = `${y}-${m}-${day}`;

    if (!dailyData[dateStr]) {
      dailyData[dateStr] = {
        date: dateStr,
        requests: 0,
        quota: 0,
        tokens: 0,
      };
    }
  }

  return Object.values(dailyData).sort((a, b) => a.date.localeCompare(b.date));
};

// 处理数据以供堆叠柱状图使用
const processModelData = () => {
  const timeData = {};
  const models = [...new Set(data.value.map((item) => item.ModelName))];

  // 填充实际数据
  data.value.forEach((item) => {
    if (!timeData[item.Day]) {
      timeData[item.Day] = { date: item.Day };
      models.forEach((model) => {
        timeData[item.Day][model] = 0;
      });
    }
    timeData[item.Day][item.ModelName] +=
      item.PromptTokens + item.CompletionTokens;
  });

  // 获取日期范围
  const dates = Object.keys(timeData);
  let maxDate = new Date();
  let minDate =
    dates.length > 0
      ? new Date(Math.min(...dates.map((d) => new Date(d))))
      : new Date();

  const dataMaxDate =
    dates.length > 0
      ? new Date(Math.max(...dates.map((d) => new Date(d))))
      : new Date();
  if (dataMaxDate > maxDate) maxDate = dataMaxDate;

  // 确保至少显示7天的数据
  const sevenDaysAgo = new Date();
  sevenDaysAgo.setDate(sevenDaysAgo.getDate() - 6); // -6是因为包含今天
  if (minDate > sevenDaysAgo) {
    minDate = sevenDaysAgo;
  }

  // 生成所有日期
  for (let d = new Date(minDate); d <= maxDate; d.setDate(d.getDate() + 1)) {
    const y = d.getFullYear();
    const m = String(d.getMonth() + 1).padStart(2, '0');
    const day = String(d.getDate()).padStart(2, '0');
    const dateStr = `${y}-${m}-${day}`;

    if (!timeData[dateStr]) {
      timeData[dateStr] = { date: dateStr };
      models.forEach((model) => {
        timeData[dateStr][model] = 0;
      });
    }
  }

  return Object.values(timeData).sort((a, b) => a.date.localeCompare(b.date));
};

// 获取所有唯一的模型名称
const getUniqueModels = () => {
  return [...new Set(data.value.map((item) => item.ModelName))];
};

// 生成随机颜色
const getRandomColor = (index) => {
  return chartConfig.barColors[index % chartConfig.barColors.length];
};

// 添加一个日期格式化函数
const formatDate = (dateStr) => {
  const date = new Date(dateStr);
  return date.toLocaleDateString('zh-CN', {
    month: 'numeric',
    day: 'numeric',
  });
};

const timeSeriesData = computed(() => processTimeSeriesData());
const modelData = computed(() => processModelData());
const models = computed(() => getUniqueModels());

const xAxisCategories = computed(() => timeSeriesData.value.map((d) => d.date));

const lineGridOption = {
  left: 10,
  right: 10,
  top: 10,
  bottom: 20,
  containLabel: false,
};

const xAxisOption = () => ({
  type: 'category',
  data: xAxisCategories.value,
  boundaryGap: false,
  axisLine: { show: false },
  axisTick: { show: false },
  axisLabel: {
    fontSize: 12,
    color: '#A3AED0',
    interval: 0,
    formatter: (value) => formatDate(value),
  },
});

const yAxisHidden = {
  type: 'value',
  show: false,
};

const tooltipStyle = {
  backgroundColor: '#fff',
  borderWidth: 0,
  borderRadius: 4,
  extraCssText: 'box-shadow: 0 2px 8px rgba(0,0,0,0.1);',
};

const buildLineOption = (dataKey, color, valueFormatter, seriesName) =>
  computed(() => ({
    grid: lineGridOption,
    xAxis: xAxisOption(),
    yAxis: yAxisHidden,
    tooltip: {
      trigger: 'axis',
      ...tooltipStyle,
      formatter: (params) => {
        const p = params[0];
        const value = valueFormatter(p.value);
        return `${t('dashboard.statistics.tooltip.date')}: ${formatDate(
          p.axisValue
        )}<br/>${seriesName()}: ${value}`;
      },
    },
    series: [
      {
        type: 'line',
        smooth: true,
        showSymbol: false,
        symbolSize: 8,
        data: timeSeriesData.value.map((d) => d[dataKey]),
        lineStyle: { width: 2, color },
        itemStyle: { color },
      },
    ],
  }));

const requestsOption = buildLineOption(
  'requests',
  chartConfig.colors.requests,
  (v) => v,
  () => t('dashboard.charts.requests.tooltip')
);

const quotaOption = buildLineOption(
  'quota',
  chartConfig.colors.quota,
  (v) => Number(v).toFixed(6),
  () => t('dashboard.charts.quota.tooltip')
);

const tokensOption = buildLineOption(
  'tokens',
  chartConfig.colors.tokens,
  (v) => v,
  () => t('dashboard.charts.tokens.tooltip')
);

const modelOption = computed(() => {
  const categories = modelData.value.map((d) => d.date);
  return {
    grid: {
      left: 10,
      right: 10,
      top: 10,
      bottom: 60,
      containLabel: true,
    },
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      ...tooltipStyle,
      formatter: (params) => {
        let lines = `${t('dashboard.statistics.tooltip.date')}: ${formatDate(
          params[0]?.axisValue
        )}`;
        params.forEach((p) => {
          lines += `<br/>${p.marker}${p.seriesName}: ${p.value}`;
        });
        return lines;
      },
    },
    legend: {
      bottom: 0,
      data: models.value,
    },
    xAxis: {
      type: 'category',
      data: categories,
      axisLine: { show: false },
      axisTick: { show: false },
      axisLabel: {
        fontSize: 12,
        color: '#A3AED0',
        interval: 0,
        formatter: (value) => formatDate(value),
      },
    },
    yAxis: {
      type: 'value',
      axisLine: { show: false },
      axisTick: { show: false },
      axisLabel: { fontSize: 12, color: '#A3AED0' },
    },
    series: models.value.map((model, index) => ({
      name: model,
      type: 'bar',
      stack: 'a',
      itemStyle: {
        color: getRandomColor(index),
        borderRadius: [4, 4, 0, 0],
      },
      data: modelData.value.map((d) => d[model] || 0),
    })),
  };
});

onMounted(() => {
  fetchDashboardData();
});
</script>

<style scoped>
.dashboard-container {
  box-sizing: border-box;
  width: 100%;
  max-width: none;
  margin: 0;
  padding: 0;
  background-color: var(--app-shell-bg, #f3f6f9);
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: var(--app-work-stack-gap, 12px);
}

.charts-grid {
  margin-bottom: 1rem;
}

.chart-card {
  height: 100%;
  box-shadow: none;
  border: none;
  border-radius: 0;
}

.chart-container {
  margin-top: 2px;
  padding: 16px;
  background-color: white;
  border-radius: 12px;
}

.header {
  color: #030d12;
  font-size: 1.2em;
  margin-bottom: 15px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-weight: 600;
  gap: 12px;
}

.echart {
  width: 100%;
}

@media (max-width: 768px) {
  .dashboard-container {
    padding: 0;
    max-width: none;
  }

  .chart-container {
    padding: 12px;
  }
}
</style>
