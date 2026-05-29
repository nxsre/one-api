<template>
  <div class="app-footer-text text-center py-3 text-xs text-gray-500">
    <div v-if="footer" class="custom-footer" v-html="footer"></div>
    <div v-else class="custom-footer">
      <a href="https://github.com/songquanpeng/one-api" target="_blank" rel="noreferrer">
        {{ systemName }} {{ version }}
      </a>
      {{ t('footer.built_by') }}
      <a href="https://github.com/songquanpeng" target="_blank" rel="noreferrer">
        {{ t('footer.built_by_name') }}
      </a>
      {{ t('footer.license') }}
      <a href="https://opensource.org/licenses/mit-license.php">{{ t('footer.mit') }}</a>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue';
import { useI18n } from 'vue-i18n';
import { getFooterHTML, getSystemName } from '../helpers';

const { t } = useI18n();
const systemName = getSystemName();
const version = import.meta.env.VITE_APP_VERSION;
const footer = ref(getFooterHTML());

let timer = null;
let remain = 5;
onMounted(() => {
  timer = setInterval(() => {
    if (remain <= 0) {
      clearInterval(timer);
      return;
    }
    remain -= 1;
    const html = localStorage.getItem('footer_html');
    if (html) footer.value = html;
  }, 200);
});
onUnmounted(() => timer && clearInterval(timer));
</script>
