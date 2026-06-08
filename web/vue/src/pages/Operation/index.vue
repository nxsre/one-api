<template>
  <div class="dashboard-container">
    <a-card class="chart-card">
      <div class="header" style="font-size: 1.2em; font-weight: 600; margin-bottom: 15px;">
        {{ t('header.operations_management') }}
      </div>
      <OperationSetting />
    </a-card>
  </div>
</template>

<script setup>
import { onMounted } from 'vue';
import { useRouter } from 'vue-router';
import { useI18n } from 'vue-i18n';
import { isRoot } from '@/helpers';
import OperationSetting from '@/components/OperationSetting.vue';

const { t } = useI18n();
const router = useRouter();

onMounted(() => {
  if (!isRoot()) {
    router.replace('/');
  }
});
</script>
