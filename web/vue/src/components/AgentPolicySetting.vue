<template>
  <a-spin :spinning="loading">
    <a-alert
      class="mb-4"
      type="info"
      show-icon
      message="全局 agent 客户端策略"
      description="按客户端类型(如 claude-code / openclaw / hermes / 普通直连)做全局放行白名单、禁用与限流。令牌 / 用户 / 租户可在各自编辑页做更细粒度配置;层级冲突时按「就近覆盖:令牌 > 用户 > 租户」生效,全局禁用优先级最高。"
    />
    <a-form layout="vertical" style="max-width: 640px">
      <AgentClientPolicyEditor v-model="policy" mode="global" />
      <a-space>
        <a-button type="primary" :loading="saving" @click="save">保存</a-button>
        <a-button @click="load">重置</a-button>
      </a-space>
    </a-form>
  </a-spin>
</template>

<script setup>
import { ref, onMounted } from 'vue';
import { API, showError, showSuccess } from '@/helpers';
import AgentClientPolicyEditor from '@/components/AgentClientPolicyEditor.vue';

const loading = ref(true);
const saving = ref(false);
const policy = ref(null);

const load = async () => {
  loading.value = true;
  try {
    const res = await API.get('/api/option/');
    const { success, data, message } = res.data;
    if (success) {
      const row = (data || []).find((it) => it.key === 'AgentClientPolicy');
      const raw = row?.value;
      policy.value = raw && raw !== '{}' ? JSON.parse(raw) : null;
    } else {
      showError(message);
    }
  } catch (e) {
    showError(e.message || '加载失败');
  } finally {
    loading.value = false;
  }
};

const save = async () => {
  saving.value = true;
  try {
    const value = policy.value ? JSON.stringify(policy.value) : '{}';
    const res = await API.put('/api/option/', { key: 'AgentClientPolicy', value });
    const { success, message } = res.data;
    if (success) {
      showSuccess('已保存全局客户端策略');
    } else {
      showError(message);
    }
  } catch (e) {
    showError(e.message || '保存失败');
  } finally {
    saving.value = false;
  }
};

onMounted(load);
</script>
