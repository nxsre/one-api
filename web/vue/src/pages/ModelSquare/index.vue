<template>
  <div class="dashboard-container">
    <a-card class="chart-card">
      <div class="ms-header">
        <h2 class="ms-title">{{ t('model_square.title') }}</h2>
        <a-input-search
          v-model:value="keyword"
          class="ms-search"
          allow-clear
          :placeholder="t('model_square.search_ph')"
          @search="reload"
        />
      </div>
      <p class="ms-subtitle">{{ t('model_square.subtitle') }}</p>

      <div class="ms-categories">
        <a-radio-group v-model:value="category" button-style="solid" @change="reload">
          <a-radio-button v-for="c in categories" :key="c.value" :value="c.value">
            {{ t(c.label) }}
          </a-radio-button>
        </a-radio-group>
      </div>

      <a-spin :spinning="loading">
        <a-empty v-if="!loading && cards.length === 0" :description="t('model_square.empty')" />
        <a-row v-else :gutter="[16, 16]" class="ms-grid">
          <a-col v-for="m in cards" :key="m.model_id" :xs="24" :sm="12" :md="8" :xl="6">
            <a-card class="ms-card" :body-style="{ padding: '14px 16px' }">
              <div class="ms-card-top">
                <div
                  class="ms-logo"
                  :style="{ background: m.brand.color, color: '#fff', borderColor: m.brand.color }"
                >
                  <span v-if="m.brand.svg" class="ms-logo-svg" v-html="m.brand.svg"></span>
                  <span v-else class="ms-logo-text">{{ m.brand.short }}</span>
                </div>
                <div class="ms-name-wrap">
                  <div class="ms-name" :title="m.display">{{ m.display }}</div>
                  <div class="ms-provider">{{ m.brand.name }}</div>
                </div>
                <a-tooltip :title="t('model_square.copy_id')">
                  <a-button type="text" size="small" class="ms-copy" @click="copyId(m.model_id)">
                    <CopyOutlined />
                  </a-button>
                </a-tooltip>
              </div>

              <div class="ms-id" :title="m.model_id">{{ m.model_id }}</div>

              <div class="ms-tags">
                <a-tag v-if="m.family" color="default">{{ m.family }}</a-tag>
                <a-tag v-for="mod in m.modalities" :key="mod" color="blue">{{ mod }}</a-tag>
                <a-tag v-if="m.reasoning" color="purple">{{ t('model_square.tag_reasoning') }}</a-tag>
                <a-tag v-if="m.tool_call" color="green">{{ t('model_square.tag_tool') }}</a-tag>
                <a-tag v-if="m.context_label" color="orange">{{ m.context_label }}</a-tag>
              </div>
            </a-card>
          </a-col>
        </a-row>
      </a-spin>

      <div v-if="!loading && cards.length" class="ms-count">
        {{ t('model_square.count', { n: cards.length }) }}
      </div>
    </a-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';
import { CopyOutlined } from '@ant-design/icons-vue';
import { API, showError, showSuccess, resolveModelBrand } from '@/helpers';

const { t } = useI18n();

const loading = ref(false);
const keyword = ref('');
const category = ref('');
const cards = ref([]);

const categories = [
  { value: '', label: 'model_square.cat_all' },
  { value: 'language', label: 'model_square.cat_language' },
  { value: 'reasoning', label: 'model_square.cat_reasoning' },
  { value: 'multimodal', label: 'model_square.cat_multimodal' },
  { value: 'code', label: 'model_square.cat_code' },
  { value: 'image', label: 'model_square.cat_image' },
];

function splitModalities(s) {
  return String(s || '')
    .split(',')
    .map((x) => x.trim())
    .filter(Boolean);
}

function contextLabel(n) {
  const v = Number(n) || 0;
  if (v <= 0) return '';
  if (v >= 1000) return `${Math.round(v / 1000)}K`;
  return String(v);
}

function decorate(item) {
  const brand = resolveModelBrand(item.model_id, item.provider_key, item.provider_display);
  const modalities = Array.from(
    new Set([...splitModalities(item.modalities_in), ...splitModalities(item.modalities_out)])
  );
  const ctx = contextLabel(item.context_limit);
  return {
    ...item,
    brand,
    display: (item.model_name && item.model_name !== item.model_id ? item.model_name : item.model_id),
    modalities,
    context_label: ctx ? `${ctx} ctx` : '',
  };
}

const reload = async () => {
  loading.value = true;
  try {
    const params = {};
    if (category.value) params.category = category.value;
    if (keyword.value.trim()) params.keyword = keyword.value.trim();
    const res = await API.get('/api/user/model_square', { params });
    const { success, message, data } = res.data;
    if (success) {
      const items = (data && data.items) || [];
      cards.value = items.map(decorate);
    } else {
      showError(message);
    }
  } catch (e) {
    showError(e.message);
  } finally {
    loading.value = false;
  }
};

const copyId = async (id) => {
  try {
    await navigator.clipboard.writeText(id);
    showSuccess(t('model_square.copied', { id }));
  } catch {
    showError(t('model_square.copy_failed'));
  }
};

onMounted(reload);
</script>

<style scoped>
.ms-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
}
.ms-title {
  margin: 0;
}
.ms-search {
  max-width: 320px;
}
.ms-subtitle {
  color: rgba(0, 0, 0, 0.45);
  margin: 6px 0 14px;
}
.ms-categories {
  margin-bottom: 18px;
}
.ms-grid {
  margin-top: 4px;
}
.ms-card {
  height: 100%;
  border-radius: 10px;
  transition: box-shadow 0.2s, transform 0.2s;
}
.ms-card:hover {
  box-shadow: 0 6px 18px rgba(0, 0, 0, 0.1);
  transform: translateY(-2px);
}
.ms-card-top {
  display: flex;
  align-items: center;
  gap: 10px;
}
.ms-logo {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  flex: 0 0 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid transparent;
  overflow: hidden;
}
.ms-logo-svg {
  width: 24px;
  height: 24px;
  display: inline-flex;
}
.ms-logo-svg :deep(svg) {
  width: 100%;
  height: 100%;
}
.ms-logo-text {
  font-weight: 700;
  font-size: 13px;
  line-height: 1;
}
.ms-name-wrap {
  min-width: 0;
  flex: 1 1 auto;
}
.ms-name {
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.ms-provider {
  font-size: 12px;
  color: rgba(0, 0, 0, 0.45);
}
.ms-copy {
  flex: 0 0 auto;
}
.ms-id {
  margin: 10px 0;
  font-family: 'SFMono-Regular', Consolas, monospace;
  font-size: 12px;
  color: rgba(0, 0, 0, 0.55);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.ms-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.ms-tags :deep(.ant-tag) {
  margin: 0;
}
.ms-count {
  margin-top: 16px;
  text-align: center;
  color: rgba(0, 0, 0, 0.45);
  font-size: 12px;
}
</style>
