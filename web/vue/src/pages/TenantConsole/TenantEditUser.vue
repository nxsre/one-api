<template>
  <div class="dashboard-container p-6">
    <a-card class="chart-card">
      <h2 class="text-xl font-semibold mb-4">
        {{ isEdit ? t('tenant_console.edit_user.title_edit') : t('tenant_console.edit_user.title_create') }}
      </h2>
      <h4 class="text-base font-medium border-b pb-2 mb-4">
        {{ t('tenant_console.edit_user.section_basic') }}
      </h4>
      <a-spin :spinning="loading">
        <a-form layout="vertical" autocomplete="off">
          <a-form-item :label="t('tenant_console.edit_user.username')" :required="!isEdit">
            <a-input
              v-model:value="inputs.username"
              :readonly="!autofillUnlocked"
              @focus="autofillUnlocked = true"
              v-bind="noAutofillTextProps"
            />
          </a-form-item>
          <a-form-item :label="t('tenant_console.edit_user.password')">
            <a-input-password
              v-model:value="inputs.password"
              :readonly="!autofillUnlocked"
              :placeholder="isEdit ? t('tenant_console.edit_user.password_unchanged_hint') : ''"
              @focus="autofillUnlocked = true"
              v-bind="noAutofillPasswordProps"
            />
          </a-form-item>
          <a-form-item :label="t('tenant_console.edit_user.display_name')">
            <a-input v-model:value="inputs.display_name" autocomplete="off" />
          </a-form-item>
          <a-form-item :label="t('tenant_console.edit_user.email')">
            <a-input v-model:value="inputs.email" autocomplete="off" />
          </a-form-item>
          <a-form-item :label="t('tenant_console.edit_user.group')">
            <a-select
              v-model:value="groupSelection"
              show-search
              mode="tags"
              :max-tag-count="1"
              :options="groupOptions"
              :field-names="{ label: 'text', value: 'value' }"
              option-filter-prop="text"
            />
          </a-form-item>
          <a-form-item :label="`${t('tenant_console.edit_user.quota')}${renderQuotaWithPrompt(inputs.quota, t)}`">
            <a-input :value="quotaText" @input="onQuotaInput" />
          </a-form-item>

          <a-form-item v-if="isEdit" :label="t('tenant_console.edit_user.status')">
            <a-select v-model:value="inputs.status" :options="statusOptions" :field-names="{ label: 'text', value: 'value' }" />
          </a-form-item>

          <a-form-item label="租户权限角色">
            <a-select v-model:value="inputs.role" :options="roleOptions" :field-names="{ label: 'text', value: 'value' }" />
          </a-form-item>
          <a-form-item v-if="inputs.role === 1" label="普通子账号额外管理权限">
            <a-select
              v-model:value="inputs.tenant_permissions"
              mode="multiple"
              :options="permissionOptions"
              :field-names="{ label: 'text', value: 'value' }"
              placeholder="(可选) 授予普通子账号特定的租户控制台管理能力"
            />
          </a-form-item>

          <h4 class="text-base font-medium border-b pb-2 mb-2">
            {{ t('tenant_console.edit_user.section_acl') }}
          </h4>
          <p class="opacity-80 mb-3">{{ t('tenant_console.edit_user.acl_hint') }}</p>
          <a-form-item :label="t('tenant_console.edit_user.allowed_models')">
            <a-select
              v-model:value="inputs.allowed_models"
              mode="multiple"
              show-search
              :options="modelOptions"
              :field-names="{ label: 'text', value: 'value' }"
              option-filter-prop="text"
              :placeholder="t('tenant_console.edit_user.allowed_models_ph')"
            />
          </a-form-item>
          <a-form-item :label="t('tenant_console.edit_user.allowed_channels')">
            <a-select
              v-model:value="inputs.allowed_channel_ids"
              mode="multiple"
              show-search
              :options="channelOptions"
              :field-names="{ label: 'text', value: 'value' }"
              option-filter-prop="text"
              :placeholder="t('tenant_console.edit_user.allowed_channels_ph')"
            />
          </a-form-item>

          <a-space>
            <a-button @click="router.push('/tenant-console/users')">
              {{ t('tenant_console.edit_user.cancel') }}
            </a-button>
            <a-button type="primary" @click="submit">
              {{ t('tenant_console.edit_user.submit') }}
            </a-button>
          </a-space>
        </a-form>
      </a-spin>
    </a-card>
  </div>
</template>

<script setup>
import { ref, reactive, computed, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter, useRoute } from 'vue-router';
import {
  API,
  showError,
  showSuccess,
  renderQuotaWithPrompt,
  noAutofillTextProps,
  noAutofillPasswordProps,
} from '@/helpers';

const { t } = useI18n();
const router = useRouter();
const route = useRoute();

const userId = computed(() => route.params.userId);
const isEdit = computed(() => Boolean(userId.value));

const loading = ref(isEdit.value);
const autofillUnlocked = ref(false);
const groupOptions = ref([]);
const modelOptions = ref([]);
const channelOptions = ref([]);

const inputs = reactive({
  user_id: '',
  username: '',
  display_name: '',
  password: '',
  email: '',
  group: 'default',
  quota: 0,
  status: 1,
  role: 1,
  allowed_models: [],
  allowed_channel_ids: [],
  tenant_permissions: [],
});

const statusOptions = [
  { value: 1, text: t('tenant_console.edit_user.status_enabled') },
  { value: 2, text: t('tenant_console.edit_user.status_disabled') },
];
const roleOptions = [
  { value: 1, text: '普通子账号 (Common)' },
  { value: 20, text: '租户管理员 (Tenant Admin)' },
];
const permissionOptions = [
  { value: 'manage_users', text: '用户管理 (manage_users)' },
  { value: 'manage_channels', text: '渠道管理 (manage_channels)' },
  { value: 'manage_tokens', text: '令牌管理 (manage_tokens)' },
  { value: 'manage_billing', text: '账单与报表 (manage_billing)' },
  { value: 'manage_nacos', text: 'Nacos 注册与配置 (manage_nacos)' },
];

const quotaText = computed(() =>
  inputs.quota === undefined || inputs.quota === null ? '' : String(inputs.quota)
);

const onQuotaInput = (e) => {
  const trimmed = String(e.target.value).trim();
  const n = trimmed === '' ? 0 : parseInt(trimmed, 10);
  if (!Number.isNaN(n)) inputs.quota = n;
};

// Group is a single string; tags-mode select allows adding a custom group.
// Keep only the last selected/typed value to mirror single-select + additions.
const groupSelection = computed({
  get: () => (inputs.group ? [inputs.group] : []),
  set: (val) => {
    if (Array.isArray(val)) {
      inputs.group = val.length ? val[val.length - 1] : '';
    } else {
      inputs.group = val || '';
    }
  },
});

const loadMeta = async () => {
  try {
    const [gr, mo] = await Promise.all([
      API.get('/api/tenant_console/meta/groups'),
      API.get('/api/tenant_console/meta/all_models'),
    ]);
    if (gr.data?.success && Array.isArray(gr.data.data)) {
      groupOptions.value = gr.data.data.map((g) => ({ key: g, text: g, value: g }));
    }
    if (mo.data?.success && Array.isArray(mo.data.data)) {
      modelOptions.value = mo.data.data.map((m) => ({ key: m.id, text: m.id, value: m.id }));
    }
  } catch (e) {
    showError(e.message);
  }
};

const loadChannelsForAcl = async (group) => {
  const g = (group && String(group).trim()) || 'default';
  try {
    const res = await API.get(
      `/api/tenant_console/meta/channels_for_acl?group=${encodeURIComponent(g)}`
    );
    if (res.data?.success && Array.isArray(res.data.data)) {
      channelOptions.value = res.data.data.map((c) => ({
        key: c.id,
        text: `#${c.id} ${c.name || ''}${
          c.tenant_id == null ? ` (${t('tenant_console.edit_user.channel_platform_badge')})` : ''
        }`,
        value: c.id,
      }));
    }
  } catch (e) {
    showError(e.message);
  }
};

const loadProfile = async () => {
  if (!isEdit.value) return;
  loading.value = true;
  try {
    const res = await API.get(
      `/api/tenant_console/users/${encodeURIComponent(userId.value)}/profile`
    );
    const { success, message, data } = res.data;
    if (!success || !data) {
      showError(message || 'load failed');
      return;
    }
    Object.assign(inputs, {
      user_id: data.user_id || '',
      username: data.username || '',
      display_name: data.display_name || '',
      password: '',
      email: data.email || '',
      group: data.group || 'default',
      quota: data.quota ?? 0,
      status: data.status ?? 1,
      role: data.role ?? 1,
      allowed_models: Array.isArray(data.allowed_models) ? data.allowed_models : [],
      allowed_channel_ids: Array.isArray(data.allowed_channel_ids) ? data.allowed_channel_ids : [],
      tenant_permissions: Array.isArray(data.tenant_permissions) ? data.tenant_permissions : [],
    });
  } catch (e) {
    showError(e.message);
  } finally {
    loading.value = false;
  }
};

const submit = async () => {
  if (!isEdit.value) {
    if (!inputs.username?.trim() || !inputs.password) {
      showError(t('tenant_console.edit_user.need_credentials'));
      return;
    }
    try {
      const res = await API.post('/api/tenant_console/users', {
        username: inputs.username.trim(),
        password: inputs.password,
        display_name: inputs.display_name?.trim() || inputs.username.trim(),
        group: inputs.group || 'default',
        quota: inputs.quota,
        role: inputs.role,
        email: inputs.email?.trim() || '',
        allowed_models: inputs.allowed_models || [],
        allowed_channel_ids: inputs.allowed_channel_ids || [],
        tenant_permissions: inputs.tenant_permissions || [],
      });
      const { success, message } = res.data;
      if (!success) {
        showError(message);
        return;
      }
      showSuccess(t('tenant_console.edit_user.create_success'));
      router.push('/tenant-console/users');
    } catch (e) {
      showError(e.message);
    }
    return;
  }

  try {
    const res = await API.put('/api/tenant_console/users', {
      user_id: inputs.user_id,
      username: inputs.username?.trim(),
      display_name: inputs.display_name?.trim(),
      password: inputs.password,
      email: inputs.email?.trim() || '',
      group: inputs.group || 'default',
      quota: inputs.quota,
      status: inputs.status,
      role: inputs.role,
      allowed_models: inputs.allowed_models || [],
      allowed_channel_ids: inputs.allowed_channel_ids || [],
      tenant_permissions: inputs.tenant_permissions || [],
    });
    const { success, message } = res.data;
    if (!success) {
      showError(message);
      return;
    }
    showSuccess(t('tenant_console.edit_user.update_success'));
    router.push('/tenant-console/users');
  } catch (e) {
    showError(e.message);
  }
};

watch(
  () => userId.value,
  () => {
    loadMeta();
    loadProfile();
  },
  { immediate: true }
);

watch(
  () => inputs.group,
  (g) => {
    loadChannelsForAcl(g);
  },
  { immediate: true }
);
</script>
