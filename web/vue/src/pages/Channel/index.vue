<template>
  <div class="dashboard-container p-6">
    <a-card class="chart-card">
      <h2 class="text-xl font-semibold mb-4">{{ t('channel.title') }}</h2>
      <ChannelsTable />
    </a-card>
  </div>
</template>

<script setup>
import { useI18n } from 'vue-i18n';
import ChannelsTable from '@/components/ChannelsTable.vue';

const { t } = useI18n();
</script>
