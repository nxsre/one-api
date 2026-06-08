<template>
  <div class="quota-amount-input flex flex-wrap items-center gap-2">
    <span class="text-sm text-gray-500">{{ t('common.quota.converter.label') }}</span>
    <a-select
      v-model:value="currency"
      style="width: 130px"
      :options="currencyOptions"
    />
    <a-input-number
      v-model:value="amount"
      :min="0"
      :step="1"
      :placeholder="t('common.quota.converter.amount_placeholder')"
      style="width: 140px"
    />
    <template v-if="currency === 'cny'">
      <span class="text-sm text-gray-500">{{ t('common.quota.converter.rate_label') }}</span>
      <a-input-number
        v-model:value="usdCnyRate"
        :min="0.01"
        :step="0.1"
        style="width: 110px"
      />
    </template>
    <span v-if="computedQuota != null" class="text-sm text-gray-400">
      = {{ computedQuota }} {{ t('common.quota.converter.unit') }}
    </span>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { getQuotaPerUnit } from '@/helpers';

const { t } = useI18n();
const emit = defineEmits(['apply']);

// 记住用户上次填写的美元-人民币汇率（系统无内置汇率，需手填）。
const RATE_KEY = 'usd_cny_rate';
const DEFAULT_RATE = 7.2;

function loadRate() {
  const raw = parseFloat(localStorage.getItem(RATE_KEY));
  return Number.isFinite(raw) && raw > 0 ? raw : DEFAULT_RATE;
}

const currency = ref('usd');
const amount = ref(null);
const usdCnyRate = ref(loadRate());

const currencyOptions = computed(() => [
  { label: t('common.quota.converter.usd'), value: 'usd' },
  { label: t('common.quota.converter.cny'), value: 'cny' },
]);

// 金额 → 额度：美元按 quota_per_unit；人民币先按汇率折算成美元再换算。
const computedQuota = computed(() => {
  const a = Number(amount.value);
  if (!Number.isFinite(a) || a <= 0) return null;
  let usd = a;
  if (currency.value === 'cny') {
    const rate = Number(usdCnyRate.value);
    if (!Number.isFinite(rate) || rate <= 0) return null;
    usd = a / rate;
  }
  return Math.round(usd * getQuotaPerUnit());
});

// 金额/币种/汇率任一变化都即时回填额度。
watch(computedQuota, (q) => {
  if (q != null) emit('apply', q);
});

watch(usdCnyRate, (rate) => {
  const n = Number(rate);
  if (Number.isFinite(n) && n > 0) {
    localStorage.setItem(RATE_KEY, String(n));
  }
});
</script>
