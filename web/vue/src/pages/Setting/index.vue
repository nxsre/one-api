<template>
  <div class="dashboard-container">
    <a-card class="chart-card">
      <h2 class="header">{{ t('setting.title') }}</h2>
      <a-tabs v-model:active-key="activeKey" class="settings-tab">
        <a-tab-pane key="personal" :tab="t('setting.tabs.personal')">
          <PersonalSetting />
        </a-tab-pane>
        <template v-if="root">
          <a-tab-pane key="system" :tab="t('setting.tabs.system')">
            <SystemSetting />
          </a-tab-pane>
          <a-tab-pane key="other" :tab="t('setting.tabs.other')">
            <OtherSetting />
          </a-tab-pane>
          <a-tab-pane key="payment" :tab="t('setting.tabs.payment')">
            <PaymentSetting />
          </a-tab-pane>
        </template>
      </a-tabs>
    </a-card>
  </div>
</template>

<script setup>
import { ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { isRoot } from '@/helpers';
import SystemSetting from '@/components/SystemSetting.vue';
import OtherSetting from '@/components/OtherSetting.vue';
import PersonalSetting from '@/components/PersonalSetting.vue';
import PaymentSetting from '@/components/PaymentSetting.vue';

const { t } = useI18n();
const root = isRoot();
const activeKey = ref('personal');
</script>

<style scoped>
.dashboard-container {
  padding: 1.5rem;
}

.chart-card .header {
  margin-bottom: 1rem;
  font-size: 1.4rem;
  font-weight: 600;
}
</style>
