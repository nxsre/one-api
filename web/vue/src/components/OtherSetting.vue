<template>
  <a-spin :spinning="loading">
    <a-form layout="vertical" @submit.prevent>
      <h3>{{ t('setting.other.notice.title') }}</h3>
      <a-form-item :label="t('setting.other.notice.content')">
        <a-textarea
          v-model:value="inputs.Notice"
          :placeholder="t('setting.other.notice.content_placeholder')"
          :rows="10"
        />
      </a-form-item>
      <a-button @click="submitNotice">
        {{ t('setting.other.notice.buttons.save') }}
      </a-button>

      <a-divider />
      <h3>{{ t('setting.other.system.title') }}</h3>
      <a-form-item :label="t('setting.other.system.name')">
        <a-input
          v-model:value="inputs.SystemName"
          :placeholder="t('setting.other.system.name_placeholder')"
          v-bind="noAutofillTextProps"
        />
      </a-form-item>
      <a-button @click="submitSystemName">
        {{ t('setting.other.system.buttons.save_name') }}
      </a-button>

      <a-form-item style="margin-top: 1rem">
        <template #label>
          {{ t('setting.other.system.theme.title') }}（<a
            href="https://github.com/songquanpeng/one-api/blob/main/web/README.md"
            target="_blank"
            >{{ t('setting.other.system.theme.link') }}</a
          >）
        </template>
        <a-input
          v-model:value="inputs.Theme"
          :placeholder="t('setting.other.system.theme.placeholder')"
          v-bind="noAutofillTextProps"
        />
      </a-form-item>
      <a-button @click="submitTheme">
        {{ t('setting.other.system.buttons.save_theme') }}
      </a-button>

      <a-form-item :label="t('setting.other.system.logo')" style="margin-top: 1rem">
        <a-input
          v-model:value="inputs.Logo"
          :placeholder="t('setting.other.system.logo_placeholder')"
          v-bind="noAutofillTextProps"
        />
      </a-form-item>
      <a-button @click="submitLogo">
        {{ t('setting.other.system.buttons.save_logo') }}
      </a-button>

      <a-divider />
      <h3>{{ t('setting.other.content.title') }}</h3>
      <a-form-item :label="t('setting.other.content.homepage.title')">
        <a-textarea
          v-model:value="inputs.HomePageContent"
          :placeholder="t('setting.other.content.homepage.placeholder')"
          :rows="14"
        />
      </a-form-item>
      <a-button @click="submitOption('HomePageContent')">
        {{ t('setting.other.content.buttons.save_homepage') }}
      </a-button>

      <a-form-item :label="t('setting.other.content.about.title')" style="margin-top: 1rem">
        <a-textarea
          v-model:value="inputs.About"
          :placeholder="t('setting.other.content.about.placeholder')"
          :rows="14"
        />
      </a-form-item>
      <a-button @click="submitAbout">
        {{ t('setting.other.content.buttons.save_about') }}
      </a-button>

      <a-alert
        type="info"
        :message="t('setting.other.copyright.notice')"
        style="margin: 1rem 0"
      />
      <a-form-item :label="t('setting.other.content.footer.title')">
        <a-textarea
          v-model:value="inputs.Footer"
          :placeholder="t('setting.other.content.footer.placeholder')"
          :rows="6"
        />
      </a-form-item>
      <a-button @click="submitOption('Footer')">
        {{ t('setting.other.content.buttons.save_footer') }}
      </a-button>
    </a-form>

    <a-modal v-model:open="showUpdateModal" :title="`新版本：${updateData.tag_name}`">
      <div v-html="updateData.content"></div>
      <template #footer>
        <a-button @click="showUpdateModal = false">关闭</a-button>
        <a-button
          @click="() => {
            showUpdateModal = false;
            openGitHubRelease();
          }"
        >
          详情
        </a-button>
      </template>
    </a-modal>
  </a-spin>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { marked } from 'marked';
import { API, showError, showSuccess, noAutofillTextProps } from '@/helpers';

const { t } = useI18n();

const inputs = reactive({
  Footer: '',
  Notice: '',
  About: '',
  SystemName: '',
  Logo: '',
  HomePageContent: '',
  Theme: '',
});
const loading = ref(false);
const showUpdateModal = ref(false);
const updateData = reactive({ tag_name: '', content: '' });

const getOptions = async () => {
  const res = await API.get('/api/option/');
  const { success, message, data } = res.data;
  if (success) {
    data.forEach((item) => {
      if (item.key in inputs) {
        inputs[item.key] = item.value;
      }
    });
  } else {
    showError(message);
  }
};

onMounted(() => {
  getOptions();
});

const updateOption = async (key, value) => {
  loading.value = true;
  const res = await API.put('/api/option/', { key, value });
  const { success, message } = res.data;
  if (success) {
    inputs[key] = value;
  } else {
    showError(message);
  }
  loading.value = false;
};

const submitNotice = () => updateOption('Notice', inputs.Notice);
const submitSystemName = () => updateOption('SystemName', inputs.SystemName);
const submitTheme = () => updateOption('Theme', inputs.Theme);
const submitLogo = () => updateOption('Logo', inputs.Logo);
const submitAbout = () => updateOption('About', inputs.About);
const submitOption = (key) => updateOption(key, inputs[key]);

const openGitHubRelease = () => {
  window.location = 'https://github.com/songquanpeng/one-api/releases/latest';
};

// eslint-disable-next-line no-unused-vars
const checkUpdate = async () => {
  const res = await API.get(
    'https://api.github.com/repos/songquanpeng/one-api/releases/latest'
  );
  const { tag_name, body } = res.data;
  if (tag_name === import.meta.env.VITE_APP_VERSION) {
    showSuccess(`已是最新版本：${tag_name}`);
  } else {
    updateData.tag_name = tag_name;
    updateData.content = marked.parse(body);
    showUpdateModal.value = true;
  }
};
</script>

<style scoped>
h3 {
  margin-top: 0.5rem;
  margin-bottom: 0.75rem;
}
</style>
