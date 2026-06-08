<template>
  <div class="agent-client-policy-editor">
    <a-form-item :label="label">
      <template #help>
        <span class="text-xs text-gray-400">
          留空表示放开所有客户端类型；选择后仅允许所选类型访问（其余拒绝）。
        </span>
      </template>
      <a-select
        v-model:value="allowed"
        mode="multiple"
        allow-clear
        show-search
        placeholder="留空 = 放开所有；可多选"
        :options="clientOptions"
        option-filter-prop="label"
        @change="emitChange"
      />
    </a-form-item>

    <a-form-item label="限流（可选）">
      <template #help>
        <span class="text-xs text-gray-400">
          对该维度匹配到的客户端类型限速：窗口期内最大请求数。两项均需 &gt; 0 才生效。
        </span>
      </template>
      <div class="flex flex-wrap items-center gap-2">
        <a-input-number
          v-model:value="maxRequests"
          :min="0"
          placeholder="最大请求数"
          style="width: 160px"
          @change="emitChange"
        />
        <span class="text-gray-400">次 /</span>
        <a-input-number
          v-model:value="windowSec"
          :min="0"
          placeholder="窗口秒数"
          style="width: 140px"
          @change="emitChange"
        />
        <span class="text-gray-400">秒</span>
      </div>
    </a-form-item>

    <a-form-item v-if="mode === 'global'" label="全局禁用的客户端类型">
      <template #help>
        <span class="text-xs text-gray-400">
          被禁用的类型在全系统内一律拒绝，优先级高于任何放行配置。
        </span>
      </template>
      <a-select
        v-model:value="disabled"
        mode="multiple"
        allow-clear
        show-search
        placeholder="选择要全局禁用的类型"
        :options="clientOptions"
        option-filter-prop="label"
        @change="emitChange"
      />
    </a-form-item>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue';
import { API, showError } from '@/helpers';

const props = defineProps({
  // 当前策略对象（与后端 agentpolicy.Policy 同构）或 null。
  modelValue: { type: Object, default: null },
  // 'simple'（令牌/用户/租户）或 'global'（管理员，含按类型禁用）。
  mode: { type: String, default: 'simple' },
  label: { type: String, default: '允许的客户端类型' },
});
const emit = defineEmits(['update:modelValue']);

const knownClients = ref([]);
const otherClient = ref('other');
const allowed = ref([]);
const disabled = ref([]);
const maxRequests = ref(null);
const windowSec = ref(null);

const clientOptions = computed(() => {
  const ids = [...knownClients.value];
  if (!ids.includes(otherClient.value)) ids.push(otherClient.value);
  return ids.map((id) => ({
    value: id,
    label: id === otherClient.value ? `${id}（普通 API/SDK 直连）` : id,
  }));
});

// 从 props 初始化本地状态。
function hydrate(p) {
  allowed.value = Array.isArray(p?.allowed_clients) ? [...p.allowed_clients] : [];
  maxRequests.value = p?.default?.max_requests || null;
  windowSec.value = p?.default?.window_sec || null;
  disabled.value = p?.rules
    ? Object.keys(p.rules).filter((k) => p.rules[k]?.disabled)
    : [];
}

// 由本地状态构建策略对象。即使全空也返回 {}（而非 null），以便清空操作能经
// GORM Updates（跳过 nil 指针）持久化，三个维度行为一致。
function buildPolicy() {
  const policy = {};
  if (allowed.value.length) policy.allowed_clients = [...allowed.value];
  if (maxRequests.value > 0 && windowSec.value > 0) {
    policy.default = { max_requests: maxRequests.value, window_sec: windowSec.value };
  }
  if (props.mode === 'global' && disabled.value.length) {
    policy.rules = {};
    for (const c of disabled.value) policy.rules[c] = { disabled: true };
  }
  return policy;
}

function emitChange() {
  emit('update:modelValue', buildPolicy());
}

const loadMeta = async () => {
  try {
    const res = await API.get('/api/agent_policy/meta');
    const { success, data, message } = res.data;
    if (success) {
      knownClients.value = Array.isArray(data.known_clients) ? data.known_clients : [];
      otherClient.value = data.other_client || 'other';
    } else if (message) {
      showError(message);
    }
  } catch (e) {
    // meta 拉取失败不阻断表单，仅无下拉候选。
  }
};

watch(() => props.modelValue, (p) => hydrate(p), { immediate: true });

onMounted(loadMeta);
</script>
