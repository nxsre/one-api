<template>
  <div style="margin-top: 8px">
    <a-alert
      type="info"
      show-icon
      class="mb-3"
      message="使用 TOTP 认证器（如 Google Authenticator）扫描二维码后，按提示输入 6 位验证码完成绑定。登录时可在验证码与备用码之间二选一。"
    />
    <a-alert
      v-if="forceMode"
      type="warning"
      show-icon
      class="mb-3"
      message="管理员已开启全员 MFA。完成两步验证配置前，你无法继续使用控制台或 API。"
    />

    <p>
      状态：<strong>{{ status.enabled ? '已启用' : '未启用' }}</strong>
      <span v-if="status.enabled && status.backup_codes_remaining != null" style="margin-left: 12px">
        剩余备用码：<strong>{{ status.backup_codes_remaining }}</strong>
      </span>
    </p>

    <a-alert
      v-if="status.locked"
      type="error"
      show-icon
      message="两步验证已锁定，请联系管理员处理。"
    />
    <template v-else-if="!status.enabled">
      <a-button v-if="forceMode" type="primary" :loading="loading" disabled>
        正在打开两步验证配置
      </a-button>
      <a-button v-else type="primary" :loading="loading" @click="startSetup">
        启用两步验证
      </a-button>
    </template>
    <template v-else>
      <a-button
        v-if="!forceMode"
        danger
        @click="() => { disableOpen = true; code = ''; confirmDisable = false; }"
      >
        禁用两步验证
      </a-button>
      <a-button
        style="margin-left: 8px"
        @click="() => { backupOpen = true; newBackupCodes = []; code = ''; }"
      >
        重新生成备用码
      </a-button>
    </template>

    <!-- Setup modal -->
    <a-modal
      v-model:open="setupOpen"
      title="设置两步验证"
      width="520px"
      :footer="null"
      :closable="!forceMode"
      :mask-closable="!forceMode"
      :keyboard="!forceMode"
    >
      <template v-if="setupData">
        <p>请使用认证器扫描以下二维码（或手动输入密钥）：</p>
        <div class="text-center">
          <img :src="qrImgUrl(setupData.qr_code_data)" alt="2FA QR" style="max-width: 220px; margin: 0 auto" />
        </div>
        <a-alert class="mt-3" type="info" show-icon>
          <template #message>
            <div style="word-break: break-all">
              <strong>密钥（勿泄露）：</strong> {{ setupData.secret }}
            </div>
          </template>
        </a-alert>
        <a-alert class="mt-3" type="warning" show-icon>
          <template #message>
            <h5 style="margin: 0 0 6px">备用码（仅显示一次，请保存）</h5>
            <pre style="white-space: pre-wrap">{{ (setupData.backup_codes || []).join('\n') }}</pre>
            <a-button size="small" @click="copy((setupData.backup_codes || []).join('\n'))">
              复制备用码
            </a-button>
          </template>
        </a-alert>
      </template>
      <div class="text-right mt-3">
        <a-button v-if="showForceCancel" :loading="cancelLoginBusy" :disabled="cancelLoginBusy || loading" @click="emitCancelLogin">
          {{ cancelLoginLabel }}
        </a-button>
        <a-button
          v-if="setupData"
          type="primary"
          style="margin-left: 8px"
          @click="() => { setupOpen = false; enableOpen = true; code = ''; }"
        >
          下一步：输入验证码启用
        </a-button>
      </div>
    </a-modal>

    <!-- Enable modal -->
    <a-modal
      v-model:open="enableOpen"
      title="确认启用"
      width="416px"
      :footer="null"
      :closable="!forceMode"
      :mask-closable="!forceMode"
      :keyboard="!forceMode"
    >
      <a-form layout="vertical">
        <a-form-item label="认证器 6 位验证码">
          <a-input v-model:value="code" placeholder="123456" @pressEnter="enable2fa" />
        </a-form-item>
      </a-form>
      <div class="text-right mt-3">
        <a-button v-if="showForceCancel" :loading="cancelLoginBusy" :disabled="cancelLoginBusy || loading" @click="emitCancelLogin">
          {{ cancelLoginLabel }}
        </a-button>
        <a-button v-else @click="enableOpen = false">取消</a-button>
        <a-button type="primary" style="margin-left: 8px" :loading="loading" @click="enable2fa">
          启用
        </a-button>
      </div>
    </a-modal>

    <!-- Disable modal -->
    <a-modal v-model:open="disableOpen" title="禁用两步验证" width="416px" :footer="null">
      <a-alert
        type="warning"
        show-icon
        class="mb-3"
        message="禁用后账户安全性降低；若管理员开启「全员 2FA」，禁用后可能无法使用部分功能。"
      />
      <a-form layout="vertical">
        <a-form-item label="验证码或备用码">
          <a-input v-model:value="code" />
        </a-form-item>
        <a-form-item>
          <a-checkbox v-model:checked="confirmDisable">我已了解禁用后果</a-checkbox>
        </a-form-item>
      </a-form>
      <div class="text-right mt-3">
        <a-button @click="disableOpen = false">取消</a-button>
        <a-button danger style="margin-left: 8px" :loading="loading" @click="disable2fa">
          确认禁用
        </a-button>
      </div>
    </a-modal>

    <!-- Backup codes modal -->
    <a-modal v-model:open="backupOpen" title="重新生成备用码" width="520px" :footer="null">
      <a-form layout="vertical">
        <a-form-item label="当前认证器 6 位验证码">
          <a-input v-model:value="code" />
        </a-form-item>
      </a-form>
      <template v-if="newBackupCodes.length > 0">
        <a-divider />
        <a-alert type="success" show-icon>
          <template #message>
            <pre style="white-space: pre-wrap">{{ newBackupCodes.join('\n') }}</pre>
            <a-button size="small" @click="copyBackups">复制新备用码</a-button>
          </template>
        </a-alert>
      </template>
      <div class="text-right mt-3">
        <a-button @click="backupOpen = false">关闭</a-button>
        <a-button type="primary" style="margin-left: 8px" :loading="loading" @click="regenBackup">
          生成
        </a-button>
      </div>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, watch } from 'vue';
import { API, copy, showError, showSuccess, showWarning } from '../helpers';

const props = defineProps({
  forceMode: Boolean,
  cancelLoginBusy: Boolean,
  cancelLoginLabel: String,
  onEnabled: Function,
});
const emit = defineEmits(['enabled', 'cancel-login']);

const cancelLoginLabel = computed(() => props.cancelLoginLabel || '取消并返回登录');

const qrImgUrl = (otpauth) =>
  `https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=${encodeURIComponent(otpauth || '')}`;

const status = reactive({
  enabled: false,
  locked: false,
  backup_codes_remaining: 0,
});
const setupOpen = ref(false);
const enableOpen = ref(false);
const disableOpen = ref(false);
const backupOpen = ref(false);
const setupData = ref(null);
const code = ref('');
const confirmDisable = ref(false);
const newBackupCodes = ref([]);
const loading = ref(false);
const statusLoaded = ref(false);
let autoSetupStarted = false;

const showForceCancel = computed(() => props.forceMode);

function setStatus(data) {
  status.enabled = !!data.enabled;
  status.locked = !!data.locked;
  status.backup_codes_remaining = data.backup_codes_remaining;
}

async function fetchStatus() {
  try {
    const res = await API.get('/api/user/2fa/status');
    if (res.data?.success && res.data.data) {
      setStatus(res.data.data);
    }
  } catch {
    showError('获取两步验证状态失败');
  } finally {
    statusLoaded.value = true;
  }
}

async function startSetup() {
  loading.value = true;
  try {
    const res = await API.post('/api/user/2fa/setup');
    if (res.data?.success) {
      setupData.value = res.data.data;
      setupOpen.value = true;
      code.value = '';
    } else {
      showError(res.data?.message || '初始化失败');
    }
  } catch {
    showError('初始化两步验证失败');
  } finally {
    loading.value = false;
  }
}

onMounted(() => {
  fetchStatus();
});

watch(
  [() => props.forceMode, () => status.enabled, () => status.locked, statusLoaded],
  () => {
    if (
      !props.forceMode ||
      !statusLoaded.value ||
      status.enabled ||
      status.locked ||
      autoSetupStarted
    ) {
      return;
    }
    autoSetupStarted = true;
    startSetup();
  }
);

async function enable2fa() {
  if (!code.value.trim()) {
    showWarning('请输入认证器上的 6 位数字验证码');
    return;
  }
  loading.value = true;
  try {
    const res = await API.post('/api/user/2fa/enable', { code: code.value.trim() });
    if (res.data?.success) {
      showSuccess('两步验证已启用');
      enableOpen.value = false;
      setupOpen.value = false;
      code.value = '';
      setupData.value = null;
      fetchStatus();
      if (typeof props.onEnabled === 'function') {
        props.onEnabled();
      }
      emit('enabled');
    } else {
      showError(res.data?.message || '启用失败');
    }
  } catch {
    showError('启用失败');
  } finally {
    loading.value = false;
  }
}

async function disable2fa() {
  if (!code.value.trim()) {
    showWarning('请输入验证码或 8 位备用码');
    return;
  }
  if (!confirmDisable.value) {
    showWarning('请勾选确认了解禁用后果');
    return;
  }
  loading.value = true;
  try {
    const res = await API.post('/api/user/2fa/disable', { code: code.value.trim() });
    if (res.data?.success) {
      showSuccess('两步验证已禁用');
      disableOpen.value = false;
      code.value = '';
      confirmDisable.value = false;
      fetchStatus();
    } else {
      showError(res.data?.message || '禁用失败');
    }
  } catch {
    showError('禁用失败');
  } finally {
    loading.value = false;
  }
}

async function regenBackup() {
  if (!code.value.trim()) {
    showWarning('请输入认证器验证码以重新生成备用码');
    return;
  }
  loading.value = true;
  try {
    const res = await API.post('/api/user/2fa/backup_codes', { code: code.value.trim() });
    if (res.data?.success && res.data.data?.backup_codes) {
      newBackupCodes.value = res.data.data.backup_codes;
      showSuccess('备用码已重新生成，请妥善保存');
      code.value = '';
      fetchStatus();
    } else {
      showError(res.data?.message || '操作失败');
    }
  } catch {
    showError('操作失败');
  } finally {
    loading.value = false;
  }
}

async function copyBackups() {
  const text = newBackupCodes.value.join('\n');
  await copy(text);
  showSuccess('已复制到剪贴板');
}

function emitCancelLogin() {
  emit('cancel-login');
}
</script>
