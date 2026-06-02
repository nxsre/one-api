<template>
  <div class="dashboard-container">
    <a-card class="chart-card">
      <h3 class="header">{{ t('user.edit.title') }}</h3>
      <a-spin :spinning="loading">
        <a-form layout="vertical" :autocomplete="noAutofillFormProps.autocomplete">
          <a-form-item :label="t('user.edit.username')">
            <a-input
              v-model:value="inputs.username"
              :placeholder="t('user.edit.username_placeholder')"
              :readonly="!autofillUnlocked"
              v-bind="noAutofillTextProps"
              @focus="autofillUnlocked = true"
            />
          </a-form-item>
          <a-form-item :label="t('user.edit.password')">
            <a-input-password
              v-model:value="inputs.password"
              :placeholder="t('user.edit.password_placeholder')"
              :readonly="!autofillUnlocked"
              v-bind="noAutofillPasswordProps"
              @focus="autofillUnlocked = true"
            />
          </a-form-item>
          <a-form-item :label="t('user.edit.display_name')">
            <a-input
              v-model:value="inputs.display_name"
              :placeholder="t('user.edit.display_name_placeholder')"
              v-bind="noAutofillTextProps"
            />
          </a-form-item>

          <template v-if="userId">
            <a-form-item :label="t('user.edit.group')">
              <a-select
                v-model:value="inputs.group"
                show-search
                :placeholder="t('user.edit.group_placeholder')"
                :options="groupSelectOptions"
                v-bind="noAutofillDropdownProps"
                :filter-option="filterGroupOption"
                @search="onGroupSearch"
              />
            </a-form-item>
            <a-form-item :label="t('user.edit.tags')">
              <a-select
                v-model:value="inputs.tags"
                mode="tags"
                :placeholder="t('user.edit.tags_placeholder')"
                :open="false"
                v-bind="noAutofillDropdownProps"
              />
            </a-form-item>
            <a-form-item :label="t('user.table.role_text')">
              <a-select
                v-model:value="inputs.role"
                :placeholder="t('user.table.filter_role')"
                :options="roleOptions"
              />
            </a-form-item>
            <a-form-item :label="quotaLabel">
              <a-input
                v-model:value="quotaText"
                inputmode="numeric"
                :placeholder="t('user.edit.quota_placeholder')"
                v-bind="noAutofillTextProps"
              />
            </a-form-item>
          </template>

          <a-form-item :label="t('user.edit.github_id')">
            <a-input
              v-model:value="inputs.github_id"
              :placeholder="t('user.edit.github_id_placeholder')"
              readonly
            />
          </a-form-item>
          <a-form-item :label="t('user.edit.wechat_id')">
            <a-input
              v-model:value="inputs.wechat_id"
              :placeholder="t('user.edit.wechat_id_placeholder')"
              readonly
            />
          </a-form-item>
          <a-form-item :label="t('user.edit.email')">
            <a-input
              v-model:value="inputs.email"
              :placeholder="t('user.edit.email_placeholder')"
              readonly
            />
          </a-form-item>

          <a-space>
            <a-button @click="handleCancel">{{ t('user.edit.buttons.cancel') }}</a-button>
            <a-button type="primary" @click="submit">{{ t('user.edit.buttons.submit') }}</a-button>
          </a-space>
        </a-form>
      </a-spin>
    </a-card>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useI18n } from 'vue-i18n';
import {
  API,
  showError,
  showSuccess,
  noAutofillFormProps,
  noAutofillPasswordProps,
  noAutofillTextProps,
  noAutofillDropdownProps,
  renderQuotaWithPrompt,
} from '@/helpers';

const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const userId = route.params.id;

const loading = ref(true);
const autofillUnlocked = ref(false);
const inputs = reactive({
  username: '',
  display_name: '',
  password: '',
  github_id: '',
  wechat_id: '',
  email: '',
  quota: 0,
  group: 'default',
  role: 1,
  tags: [],
});
const groupOptions = ref([]);
const groupSearch = ref('');

const roleOptions = computed(() => [
  { value: 1, label: t('user.table.role_types.normal') },
  { value: 10, label: t('user.table.role_types.admin') },
  { value: 20, label: t('user.table.role_types.tenant_admin') },
  { value: 50, label: t('user.table.role_types.super_admin') },
  { value: 100, label: t('user.table.role_types.root') },
]);

// Mirror Semantic UI's `allowAdditions`: expose a synthetic option for free text.
const groupSelectOptions = computed(() => {
  const opts = groupOptions.value.map((g) => ({ value: g, label: g }));
  const search = groupSearch.value.trim();
  if (search && !groupOptions.value.includes(search)) {
    opts.push({ value: search, label: `${t('user.edit.group_addition')} ${search}` });
  }
  return opts;
});

const filterGroupOption = (input, option) =>
  String(option.value).toLowerCase().includes(String(input).toLowerCase());

const onGroupSearch = (value) => {
  groupSearch.value = value;
};

const quotaLabel = computed(() => `${t('user.edit.quota')}${renderQuotaWithPrompt(inputs.quota, t)}`);

const quotaText = computed({
  get() {
    return inputs.quota === undefined || inputs.quota === null ? '' : String(inputs.quota);
  },
  set(value) {
    const trimmed = String(value).trim();
    const n = trimmed === '' ? 0 : parseInt(trimmed, 10);
    if (!Number.isNaN(n)) {
      inputs.quota = n;
    }
  },
});

// Only allow same-origin relative redirects (no open redirect).
function consumeSafeInternalRedirect() {
  const raw = route.query.redirect;
  if (!raw) return null;
  let decoded;
  try {
    decoded = decodeURIComponent(String(raw).trim());
  } catch {
    return null;
  }
  if (!decoded.startsWith('/') || decoded.startsWith('//')) return null;
  return decoded;
}

const fetchGroups = async () => {
  try {
    const res = await API.get('/api/group/');
    groupOptions.value = res.data.data;
  } catch (error) {
    showError(error.message);
  }
};

const handleCancel = () => {
  if (userId) {
    router.push('/user');
    return;
  }
  const back = consumeSafeInternalRedirect();
  if (back) {
    router.push(back);
    return;
  }
  router.push('/setting');
};

const loadUser = async () => {
  let res;
  if (userId) {
    res = await API.get(`/api/user/${userId}`);
  } else {
    res = await API.get('/api/user/self');
  }
  const { success, message, data } = res.data;
  if (success) {
    data.password = '';
    Object.assign(inputs, data);
    if (!Array.isArray(inputs.tags)) inputs.tags = [];
  } else {
    showError(message);
  }
  loading.value = false;
};

onMounted(() => {
  loadUser();
  if (userId) {
    fetchGroups();
  }
});

const submit = async () => {
  let res;
  if (userId) {
    const data = { ...inputs, user_id: userId };
    if (typeof data.quota === 'string') {
      data.quota = parseInt(data.quota);
    }
    res = await API.put('/api/user/', data);
  } else {
    res = await API.put('/api/user/self', { ...inputs });
  }
  const { success, message } = res.data;
  if (success) {
    showSuccess(t('user.messages.update_success'));
    if (!userId) {
      const back = consumeSafeInternalRedirect();
      if (back) {
        window.location.assign(back);
      }
    }
  } else {
    showError(message);
  }
};
</script>
