<template>
  <a-button
    type="text"
    size="small"
    :title="dark ? t('header.theme_switch_to_light') : t('header.theme_switch_to_dark')"
    @click="onClick"
  >
    <template #icon>
      <BulbOutlined v-if="dark" />
      <BulbFilled v-else />
    </template>
  </a-button>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue';
import { useI18n } from 'vue-i18n';
import { BulbOutlined, BulbFilled } from '@ant-design/icons-vue';
import { getNacosTheme, toggleNacosTheme } from '../helpers/nacosTheme';

const { t } = useI18n();
const dark = ref(getNacosTheme() === 'dark');

const sync = () => {
  dark.value = getNacosTheme() === 'dark';
};

function onClick() {
  const next = toggleNacosTheme();
  dark.value = next === 'dark';
}

onMounted(() => {
  window.addEventListener('storage', sync);
  window.addEventListener('one-api:nacos-theme-changed', sync);
});
onUnmounted(() => {
  window.removeEventListener('storage', sync);
  window.removeEventListener('one-api:nacos-theme-changed', sync);
});

defineProps({ compact: Boolean });
</script>
