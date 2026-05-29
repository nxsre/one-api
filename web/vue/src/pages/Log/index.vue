<template>
  <div class="dashboard-container">
    <a-card class="chart-card">
      <div class="header" style="font-size: 1.28em; font-weight: 700; margin-bottom: 0.75rem">
        {{ t('log.title') }}
      </div>
      <LogsTable />
    </a-card>
  </div>
</template>

<script setup>
import { useI18n } from 'vue-i18n';
import LogsTable from '@/components/LogsTable.vue';

const { t } = useI18n();
</script>
