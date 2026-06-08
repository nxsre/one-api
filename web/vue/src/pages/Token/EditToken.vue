<template>
  <div class="dashboard-container p-6">
    <a-card class="chart-card">
      <h2 class="text-xl font-semibold mb-4">
        {{ isEdit ? t('token.edit.title_edit') : t('token.edit.title_create') }}
      </h2>
      <a-spin :spinning="loading">
        <a-form layout="vertical" autocomplete="off">
          <a-form-item :label="t('token.edit.name')">
            <a-input
              v-model:value="inputs.name"
              :placeholder="t('token.edit.name_placeholder')"
              autocomplete="off"
            />
          </a-form-item>

          <a-form-item :label="t('token.edit.models')">
            <a-select
              v-model:value="inputs.models"
              mode="multiple"
              show-search
              allow-clear
              :placeholder="t('token.edit.models_placeholder')"
              :options="modelOptions"
              :field-names="{ label: 'text', value: 'value' }"
              option-filter-prop="text"
            />
          </a-form-item>

          <AgentClientPolicyEditor v-model="inputs.agent_client_policy" />

          <a-form-item :label="t('token.edit.bulk_channels_label')">
            <div class="flex flex-wrap items-start gap-2">
              <a-select
                v-model:value="bulkChannelIds"
                mode="multiple"
                show-search
                allow-clear
                style="flex: 1 1 280px; min-width: 200px"
                :placeholder="t('token.edit.bulk_channels_placeholder')"
                :options="bulkChannelOptions"
                :field-names="{ label: 'text', value: 'value' }"
                option-filter-prop="text"
              />
              <a-button :disabled="!bulkChannelIds.length" @click="applyBulkChannels">
                {{ t('token.edit.bulk_channels_button') }}
              </a-button>
            </div>
          </a-form-item>

          <a-form-item>
            <template #label>
              <div>
                <div>{{ t('token.edit.ip_limit') }}</div>
                <div class="text-xs text-gray-400">{{ t('token.edit.ip_limit_placeholder') }}</div>
              </div>
            </template>
            <MonacoEditor v-model:value="inputs.subnet" language="plaintext" :height="120" />
          </a-form-item>

          <a-form-item :label="t('token.edit.expire_time')">
            <a-input
              v-model:value="inputs.expired_time"
              type="datetime-local"
              :placeholder="t('token.edit.expire_time_placeholder')"
              autocomplete="off"
            />
          </a-form-item>

          <div class="flex flex-wrap gap-2 mb-4">
            <a-button @click="setExpiredTime(0, 0, 0, 0)">{{ t('token.edit.buttons.never_expire') }}</a-button>
            <a-button @click="setExpiredTime(1, 0, 0, 0)">{{ t('token.edit.buttons.expire_1_month') }}</a-button>
            <a-button @click="setExpiredTime(0, 1, 0, 0)">{{ t('token.edit.buttons.expire_1_day') }}</a-button>
            <a-button @click="setExpiredTime(0, 0, 1, 0)">{{ t('token.edit.buttons.expire_1_hour') }}</a-button>
            <a-button @click="setExpiredTime(0, 0, 0, 1)">{{ t('token.edit.buttons.expire_1_minute') }}</a-button>
          </div>

          <a-alert class="mb-4" :message="t('token.edit.quota_notice')" />

          <a-form-item :label="`${t('token.edit.quota')}${renderQuotaWithPrompt(inputs.remain_quota, t)}`">
            <a-input
              :value="String(inputs.remain_quota ?? '')"
              :readonly="inputs.unlimited_quota"
              :placeholder="t('token.edit.quota_placeholder')"
              autocomplete="off"
              @input="onQuotaInput"
            />
          </a-form-item>

          <QuotaAmountInput
            v-if="!inputs.unlimited_quota"
            class="mb-4"
            @apply="(q) => (inputs.remain_quota = q)"
          />

          <div class="flex flex-wrap justify-between gap-2">
            <a-button @click="setUnlimitedQuota">
              {{ inputs.unlimited_quota ? t('token.edit.buttons.cancel_unlimited') : t('token.edit.buttons.unlimited_quota') }}
            </a-button>
            <div class="flex gap-2">
              <a-button @click="handleCancel">{{ t('token.edit.buttons.cancel') }}</a-button>
              <a-button type="primary" @click="submit">{{ t('token.edit.buttons.submit') }}</a-button>
            </div>
          </div>
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
  buildTokenModelOptionsFromDetail,
  distinctChannelsFromModelDetailItems,
  showError,
  showSuccess,
  timestamp2string,
  renderQuotaWithPrompt,
} from '@/helpers';
import MonacoEditor from '@/components/MonacoEditor.vue';
import QuotaAmountInput from '@/components/QuotaAmountInput.vue';
import AgentClientPolicyEditor from '@/components/AgentClientPolicyEditor.vue';

const { t } = useI18n();
const router = useRouter();
const route = useRoute();

const tokenId = computed(() => route.params.id);
const isEdit = computed(() => tokenId.value !== undefined);

const buildOriginInputs = () => ({
  name: '',
  remain_quota: isEdit.value ? 0 : 500000,
  expired_time: -1,
  unlimited_quota: false,
  models: [],
  subnet: '',
  agent_client_policy: null,
});

const loading = ref(isEdit.value);
const modelOptions = ref([]);
const modelDetailItems = ref([]);
const bulkChannelIds = ref([]);
const inputs = reactive(buildOriginInputs());

const bulkChannelOptions = computed(() =>
  distinctChannelsFromModelDetailItems(modelDetailItems.value)
);

function resetInputs(source) {
  Object.assign(inputs, buildOriginInputs(), source || {});
}

const handleCancel = () => {
  router.push('/token');
};

const setExpiredTime = (month, day, hour, minute) => {
  let now = new Date();
  let timestamp = now.getTime() / 1000;
  let seconds = month * 30 * 24 * 60 * 60;
  seconds += day * 24 * 60 * 60;
  seconds += hour * 60 * 60;
  seconds += minute * 60;
  if (seconds !== 0) {
    timestamp += seconds;
    inputs.expired_time = timestamp2string(timestamp);
  } else {
    inputs.expired_time = -1;
  }
};

const setUnlimitedQuota = () => {
  inputs.unlimited_quota = !inputs.unlimited_quota;
};

const onQuotaInput = (e) => {
  if (inputs.unlimited_quota) return;
  const v = e.target.value;
  const n = v.trim() === '' ? 0 : parseInt(v, 10);
  inputs.remain_quota = Number.isNaN(n) ? inputs.remain_quota : n;
};

const applyBulkChannels = () => {
  const ids = new Set(bulkChannelIds.value || []);
  const next = new Set(inputs.models || []);
  for (const row of modelDetailItems.value) {
    if (ids.has(row.channel_id)) next.add(row.model);
  }
  inputs.models = Array.from(next).sort((a, b) =>
    String(a).localeCompare(String(b))
  );
};

const loadPage = async () => {
  loading.value = isEdit.value;
  try {
    const detailReq = API.get(`/api/user/available_models_detail`);
    let parsedModels = [];
    if (isEdit.value) {
      const tr = await API.get(`/api/token/${tokenId.value}`);
      const trBody = tr.data || {};
      if (trBody.success && trBody.data) {
        const data = { ...trBody.data };
        if (data.expired_time !== -1) {
          data.expired_time = timestamp2string(data.expired_time);
        }
        if (data.models === '' || data.models == null) {
          data.models = [];
        } else if (typeof data.models === 'string') {
          data.models = data.models.split(',').map((s) => s.trim()).filter(Boolean);
        }
        parsedModels = Array.isArray(data.models) ? data.models : [];
        resetInputs(data);
      } else {
        showError(trBody.message || 'Failed to load token');
      }
    }
    const dr = await detailReq;
    const dBody = dr.data || {};
    const items = Array.isArray(dBody.data?.items) ? dBody.data.items : [];
    if (dBody.success) {
      modelDetailItems.value = items;
      modelOptions.value = buildTokenModelOptionsFromDetail(items, parsedModels, t);
    } else {
      showError(dBody.message || 'Failed to load models');
    }
  } catch (error) {
    showError(error.message || 'Network error');
  }
  loading.value = false;
};

const submit = async () => {
  if (!isEdit.value && inputs.name === '') return;
  let localInputs = { ...inputs };
  localInputs.remain_quota = parseInt(localInputs.remain_quota);
  if (localInputs.expired_time !== -1) {
    let time = Date.parse(localInputs.expired_time);
    if (isNaN(time)) {
      showError(t('token.edit.messages.expire_time_invalid'));
      return;
    }
    localInputs.expired_time = Math.ceil(time / 1000);
  }
  localInputs.models = (localInputs.models || []).join(',');
  let res;
  if (isEdit.value) {
    res = await API.put(`/api/token/`, {
      ...localInputs,
      id: parseInt(tokenId.value),
    });
  } else {
    res = await API.post(`/api/token/`, localInputs);
  }
  const { success, message } = res.data;
  if (success) {
    if (isEdit.value) {
      showSuccess(t('token.edit.messages.update_success'));
    } else {
      showSuccess(t('token.edit.messages.create_success'));
      resetInputs();
      bulkChannelIds.value = [];
    }
  } else {
    showError(message);
  }
};

watch(
  () => tokenId.value,
  () => {
    loadPage().catch((error) => {
      showError(error.message || 'Failed to load page');
      loading.value = false;
    });
  },
  { immediate: true }
);
</script>
