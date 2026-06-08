<template>
  <div class="dashboard-container">
    <a-card class="chart-card">
      <h3 class="header">{{ t('user.title') }}</h3>
      <UsersTable />
    </a-card>
  </div>
</template>

<script setup>
import { useI18n } from 'vue-i18n';
import UsersTable from '@/components/UsersTable.vue';

const { t } = useI18n();
</script>
