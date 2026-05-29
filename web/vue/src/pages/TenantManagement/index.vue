<template>
  <div class="dashboard-container p-6">
    <a-card class="chart-card">
      <h2 class="text-xl font-semibold mb-4">租户管理</h2>
      <div class="mt-2 mb-2">
        <a-button size="small" type="primary" @click="createModalOpen = true">新建租户</a-button>
      </div>
      <a-table
        row-key="id"
        :columns="columns"
        :data-source="tenants"
        :pagination="false"
        size="middle"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'discount_ratio'">{{ record.discount_ratio || 1.0 }}</template>
          <template v-else-if="column.key === 'price_per_1k_api_call'">
            {{ record.price_per_1k_api_call !== undefined && record.price_per_1k_api_call >= 0 ? record.price_per_1k_api_call : '默认' }}
          </template>
          <template v-else-if="column.key === 'created_time'">
            {{ timestamp2string(record.created_time) }}
          </template>
          <template v-else-if="column.key === 'actions'">
            <a-space>
              <a-button size="small" @click="openEditModal(record)">编辑兜底计费</a-button>
              <a-button size="small" @click="openBillingRulesModal(record)">时效规则</a-button>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 新建租户 -->
    <a-modal v-model:open="createModalOpen" title="新建租户" :footer="null" @cancel="createModalOpen = false">
      <a-spin :spinning="createSubmitting">
        <a-form layout="vertical" autocomplete="off">
          <a-form-item label="租户名称" required>
            <a-input v-model:value="createInputs.name" placeholder="例如：某某企业" />
          </a-form-item>
          <a-form-item label="标识 (Slug)" required>
            <a-input v-model:value="createInputs.slug" placeholder="例如：my-company" />
          </a-form-item>
          <a-form-item label="备注">
            <a-input v-model:value="createInputs.remark" />
          </a-form-item>
          <div class="flex gap-3">
            <a-form-item label="租户管理员账号" required class="flex-1">
              <a-input
                v-model:value="createInputs.admin_username"
                placeholder="登录用户名"
                :readonly="!createAutofillUnlocked"
                autocomplete="off"
                @focus="createAutofillUnlocked = true"
              />
            </a-form-item>
            <a-form-item label="管理员密码" required class="flex-1">
              <a-input-password
                v-model:value="createInputs.admin_password"
                :readonly="!createAutofillUnlocked"
                autocomplete="new-password"
                @focus="createAutofillUnlocked = true"
              />
            </a-form-item>
          </div>
          <a-form-item label="管理员显示名">
            <a-input v-model:value="createInputs.admin_display_name" placeholder="可选，默认同账号名" />
          </a-form-item>
        </a-form>
      </a-spin>
      <div class="flex justify-end gap-2 mt-2">
        <a-button @click="createModalOpen = false">取消</a-button>
        <a-button type="primary" :loading="createSubmitting" @click="submitCreate">创建</a-button>
      </div>
    </a-modal>

    <!-- 编辑兜底计费 -->
    <a-modal v-model:open="editModalOpen" title="编辑租户计费配置" :footer="null" width="420px" @cancel="editModalOpen = false">
      <a-form layout="vertical">
        <a-form-item label="全局折扣率 (如 0.8 表示 8 折)">
          <a-input v-model:value="editInputs.discount_ratio" type="number" step="0.1" />
        </a-form-item>
        <a-form-item label="私有渠道默认千次调用单价 (-1 表示使用系统默认配置)">
          <a-input v-model:value="editInputs.price_per_1k_api_call" type="number" step="0.001" />
        </a-form-item>
      </a-form>
      <div class="flex justify-end gap-2 mt-2">
        <a-button @click="editModalOpen = false">取消</a-button>
        <a-button type="primary" @click="submitEdit">保存</a-button>
      </div>
    </a-modal>

    <!-- 时效计费规则 -->
    <a-modal
      v-model:open="billingRulesModalOpen"
      :title="`管理时效计费规则 - ${editingTenant?.name || ''}`"
      :footer="null"
      width="900px"
      @cancel="onCloseBillingRules"
    >
      <a-form layout="vertical">
        <div class="grid grid-cols-3 gap-3">
          <a-form-item label="规则类型">
            <a-select
              :value="billingRuleInputs.rule_type"
              :options="ruleTypeOptions"
              :field-names="{ label: 'text', value: 'value' }"
              @change="onRuleTypeChange"
            />
          </a-form-item>
          <a-form-item label="渠道（可搜索；选「全部」表示全局）">
            <a-select
              v-model:value="channelIdModel"
              show-search
              placeholder="选择或输入搜索"
              :options="channelDropdownOptions"
              :field-names="{ label: 'text', value: 'value' }"
              option-filter-prop="text"
            />
          </a-form-item>
          <a-form-item label="设定值">
            <a-input v-model:value="billingRuleInputs.value" type="number" step="0.01" />
          </a-form-item>
        </div>
        <div class="grid grid-cols-4 gap-3 items-end">
          <a-form-item label="生效时间">
            <a-input v-model:value="billingRuleInputs.start_time" type="datetime-local" />
          </a-form-item>
          <a-form-item label="失效时间">
            <a-input v-model:value="billingRuleInputs.end_time" type="datetime-local" />
          </a-form-item>
          <a-form-item label="状态">
            <a-select
              v-model:value="billingRuleInputs.status"
              :options="statusOptions"
              :field-names="{ label: 'text', value: 'value' }"
            />
          </a-form-item>
          <a-form-item label="&nbsp;">
            <a-space>
              <a-button type="primary" @click="submitBillingRule">
                {{ billingRuleEditingId != null ? '保存修改' : '添加' }}
              </a-button>
              <a-button v-if="billingRuleEditingId != null" @click="cancelBillingRuleEdit">
                取消编辑
              </a-button>
            </a-space>
          </a-form-item>
        </div>
      </a-form>

      <a-table
        row-key="id"
        :columns="ruleColumns"
        :data-source="billingRules"
        :pagination="false"
        size="middle"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'rule_type'">{{ record.rule_type === 1 ? '折扣率' : '单价' }}</template>
          <template v-else-if="column.key === 'channel_id'">{{ record.channel_id === 0 ? '全局' : record.channel_id }}</template>
          <template v-else-if="column.key === 'start_time'">{{ timestamp2string(record.start_time) }}</template>
          <template v-else-if="column.key === 'end_time'">{{ timestamp2string(record.end_time) }}</template>
          <template v-else-if="column.key === 'status'">{{ record.status === 1 ? '启用' : '禁用' }}</template>
          <template v-else-if="column.key === 'actions'">
            <a-space>
              <a-button size="small" @click="startEditBillingRule(record)">编辑</a-button>
              <a-button size="small" danger @click="deleteBillingRule(record.id)">删除</a-button>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, reactive, computed, watch, onMounted } from 'vue';
import { API, showError, showSuccess, timestamp2string } from '@/helpers';

/** Unix 秒 -> datetime-local 字符串（本地时区） */
function tsToDatetimeLocal(ts) {
  if (ts == null || Number.isNaN(Number(ts))) return '';
  const d = new Date(Number(ts) * 1000);
  const pad = (n) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

const tenants = ref([]);
const createAutofillUnlocked = ref(false);

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id' },
  { title: '企业名称', dataIndex: 'name', key: 'name' },
  { title: '标识 (Slug)', dataIndex: 'slug', key: 'slug' },
  { title: '全局折扣', key: 'discount_ratio' },
  { title: '私有渠道千次调用单价', key: 'price_per_1k_api_call' },
  { title: '创建时间', key: 'created_time' },
  { title: '操作', key: 'actions' },
];

const ruleColumns = [
  { title: 'ID', dataIndex: 'id', key: 'id' },
  { title: '类型', key: 'rule_type' },
  { title: '渠道', key: 'channel_id' },
  { title: '数值', dataIndex: 'value', key: 'value' },
  { title: '生效时间', key: 'start_time' },
  { title: '失效时间', key: 'end_time' },
  { title: '状态', key: 'status' },
  { title: '操作', key: 'actions' },
];

const ruleTypeOptions = [
  { value: 1, text: '平台渠道折扣率 (如0.8)' },
  { value: 2, text: '私有渠道千次调用单价' },
];
const statusOptions = [
  { value: 1, text: '启用' },
  { value: 2, text: '禁用' },
];

// edit billing
const editModalOpen = ref(false);
const editingTenant = ref(null);
const editInputs = reactive({
  discount_ratio: 1.0,
  price_per_1k_api_call: -1.0,
});

// billing rules
const billingRulesModalOpen = ref(false);
const billingRules = ref([]);
const billingRuleEditingId = ref(null);
const billingChannelList = ref([]);
const billingRuleInputs = reactive({
  channel_id: '0',
  rule_type: 1,
  value: '',
  start_time: '',
  end_time: '',
  status: 1,
});

const channelIdModel = computed({
  get: () => String(billingRuleInputs.channel_id),
  set: (v) => {
    billingRuleInputs.channel_id = String(v);
  },
});

// create
const createModalOpen = ref(false);
const createSubmitting = ref(false);
const createInputs = reactive({
  name: '',
  slug: '',
  remark: '',
  admin_username: '',
  admin_password: '',
  admin_display_name: '',
});

const channelDropdownOptions = computed(() => {
  const all = [{ key: 'ch-0', text: '全部（全局）', value: '0' }];
  const fromApi = (billingChannelList.value || []).map((ch) => ({
    key: `ch-${ch.id}`,
    text: `${ch.id} · ${ch.name || ''}`,
    value: String(ch.id),
  }));
  const ids = new Set(fromApi.map((o) => o.value));
  const sel = String(billingRuleInputs.channel_id ?? '0');
  if (sel !== '0' && !ids.has(sel)) {
    fromApi.unshift({
      key: `ch-missing-${sel}`,
      text: `${sel}（当前规则，列表中未找到该渠道）`,
      value: sel,
    });
  }
  return [...all, ...fromApi];
});

const resetBillingRuleFormDraft = (ruleType) => ({
  channel_id: '0',
  rule_type: Number(ruleType),
  value: '',
  start_time: '',
  end_time: '',
  status: 1,
});

const loadTenants = async () => {
  const res = await API.get('/api/platform/tenants');
  const { success, message, data } = res.data;
  if (success) {
    tenants.value = data || [];
  } else {
    showError(message);
  }
};

const submitCreate = async () => {
  if (!createInputs.name || !createInputs.slug || !createInputs.admin_username || !createInputs.admin_password) {
    showError('名称、Slug、管理员账号及密码均不能为空');
    return;
  }
  createSubmitting.value = true;
  try {
    const res = await API.post('/api/platform/tenants', { ...createInputs });
    const { success, message } = res.data;
    if (success) {
      showSuccess('租户创建成功');
      createModalOpen.value = false;
      Object.assign(createInputs, {
        name: '',
        slug: '',
        remark: '',
        admin_username: '',
        admin_password: '',
        admin_display_name: '',
      });
      await loadTenants();
    } else {
      showError(message);
    }
  } catch (e) {
    showError(e.message);
  } finally {
    createSubmitting.value = false;
  }
};

const openEditModal = (tenant) => {
  editingTenant.value = tenant;
  editInputs.discount_ratio = tenant.discount_ratio || 1.0;
  editInputs.price_per_1k_api_call =
    tenant.price_per_1k_api_call !== undefined ? tenant.price_per_1k_api_call : -1.0;
  editModalOpen.value = true;
};

const submitEdit = async () => {
  const res = await API.put(`/api/platform/tenants/${editingTenant.value.id}/billing`, {
    discount_ratio: parseFloat(editInputs.discount_ratio),
    price_per_1k_api_call: parseFloat(editInputs.price_per_1k_api_call),
  });
  if (res.data.success) {
    showSuccess('保存成功');
    editModalOpen.value = false;
    loadTenants();
  } else {
    showError(res.data.message);
  }
};

const loadBillingRules = async (tenantId) => {
  const res = await API.get(`/api/platform/tenants/${tenantId}/billing_rules`);
  if (res.data.success) {
    billingRules.value = res.data.data || [];
  }
};

const loadBillingChannels = async () => {
  if (!billingRulesModalOpen.value || !editingTenant.value) return;
  const tid = billingRuleInputs.rule_type === 2 ? editingTenant.value.id : null;
  const baseUrl = '/api/channel/?p=0&page_size=500';
  const url = tid != null ? `${baseUrl}&tenant_id=${tid}` : baseUrl;
  try {
    const res = await API.get(url);
    if (res.data.success) {
      billingChannelList.value = res.data.data || [];
    }
  } catch {
    billingChannelList.value = [];
  }
};

const openBillingRulesModal = async (tenant) => {
  editingTenant.value = tenant;
  billingRuleEditingId.value = null;
  Object.assign(billingRuleInputs, resetBillingRuleFormDraft(1));
  billingRulesModalOpen.value = true;
  await loadBillingRules(tenant.id);
};

const onCloseBillingRules = () => {
  billingRulesModalOpen.value = false;
  billingRuleEditingId.value = null;
};

const onRuleTypeChange = (value) => {
  Object.assign(billingRuleInputs, resetBillingRuleFormDraft(value));
  billingRuleEditingId.value = null;
};

const cancelBillingRuleEdit = () => {
  billingRuleEditingId.value = null;
  Object.assign(billingRuleInputs, resetBillingRuleFormDraft(billingRuleInputs.rule_type));
};

const startEditBillingRule = (r) => {
  billingRuleEditingId.value = r.id;
  Object.assign(billingRuleInputs, {
    channel_id: String(r.channel_id ?? 0),
    rule_type: r.rule_type,
    value: r.value !== undefined && r.value !== null ? String(r.value) : '',
    start_time: tsToDatetimeLocal(r.start_time),
    end_time: tsToDatetimeLocal(r.end_time),
    status: r.status,
  });
};

const submitBillingRule = async () => {
  if (!billingRuleInputs.value || !billingRuleInputs.start_time || !billingRuleInputs.end_time) {
    showError('请填写完整信息');
    return;
  }
  const body = {
    channel_id: parseInt(billingRuleInputs.channel_id || 0, 10),
    rule_type: parseInt(billingRuleInputs.rule_type, 10),
    value: parseFloat(billingRuleInputs.value),
    start_time: Math.floor(new Date(billingRuleInputs.start_time).getTime() / 1000),
    end_time: Math.floor(new Date(billingRuleInputs.end_time).getTime() / 1000),
    status: parseInt(billingRuleInputs.status, 10),
  };
  let res;
  if (billingRuleEditingId.value != null) {
    res = await API.put(
      `/api/platform/tenants/${editingTenant.value.id}/billing_rules/${billingRuleEditingId.value}`,
      body
    );
  } else {
    res = await API.post(`/api/platform/tenants/${editingTenant.value.id}/billing_rules`, body);
  }
  if (res.data.success) {
    showSuccess(billingRuleEditingId.value != null ? '更新成功' : '添加成功');
    billingRuleEditingId.value = null;
    Object.assign(billingRuleInputs, resetBillingRuleFormDraft(billingRuleInputs.rule_type));
    loadBillingRules(editingTenant.value.id);
  } else {
    showError(res.data.message);
  }
};

const deleteBillingRule = async (ruleId) => {
  const res = await API.delete(
    `/api/platform/tenants/${editingTenant.value.id}/billing_rules/${ruleId}`
  );
  if (res.data.success) {
    showSuccess('删除成功');
    if (billingRuleEditingId.value === ruleId) {
      cancelBillingRuleEdit();
    }
    loadBillingRules(editingTenant.value.id);
  } else {
    showError(res.data.message);
  }
};

watch(
  () => [billingRulesModalOpen.value, editingTenant.value, billingRuleInputs.rule_type],
  () => {
    loadBillingChannels();
  }
);

onMounted(() => {
  loadTenants();
});
</script>
