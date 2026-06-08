<template>
  <div class="dashboard-container">
    <a-card class="chart-card">
      <h3 class="header">{{ t('nacos.perm_title') }}</h3>
      <a-alert type="warning" :message="t('nacos.perm_hint')" show-icon style="margin: 8px 0 16px" />

      <div style="display: flex; flex-wrap: wrap; align-items: center; gap: 8px 10px; margin-bottom: 16px">
        <a-input
          :placeholder="t('nacos.perm_search_placeholder')"
          v-model:value="keyword"
          style="min-width: 200px; flex: 1 1 220px; margin-bottom: 0"
          @keydown.enter.prevent="searchUsers"
        />
        <a-button type="primary" :loading="searching" :disabled="searching" @click="searchUsers">
          {{ t('nacos.perm_search_btn') }}
        </a-button>
        <a-button size="small" :disabled="!searchResults.length" @click="selectAllInResults">
          {{ t('nacos.perm_select_all') }}
        </a-button>
        <a-button size="small" :disabled="selectedCount === 0" @click="clearSelection">
          {{ t('nacos.perm_select_none') }}
        </a-button>
        <span style="color: #666; font-size: 0.95em">
          {{ t('nacos.perm_selected_count', { count: selectedCount }) }}
        </span>
      </div>

      <a-table
        v-if="searchResults.length > 0"
        size="small"
        :columns="userColumns"
        :data-source="searchResults"
        :row-key="(u) => u.user_id"
        :pagination="false"
        :row-class-name="(u) => (selectedIds.has(u.user_id) ? 'ant-table-row-selected' : '')"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'select'">
            <a-checkbox :checked="selectedIds.has(record.user_id)" @change="() => toggleUser(record.user_id)" />
          </template>
          <template v-else-if="column.key === 'id'">{{ record.user_id }}</template>
          <template v-else-if="column.key === 'username'">{{ record.username }}</template>
          <template v-else-if="column.key === 'display'">{{ record.display_name || '—' }}</template>
          <template v-else-if="column.key === 'email'">{{ record.email || '—' }}</template>
          <template v-else-if="column.key === 'role'">{{ roleLabel(record.role) }}</template>
        </template>
      </a-table>

      <div style="margin-top: 16px; margin-bottom: 8px">
        <a-button :disabled="loading || selectedCount !== 1" @click="loadAcl">
          {{ t('nacos.load_acl') }}
        </a-button>
        <a-button type="primary" :disabled="loading || selectedCount < 1" @click="saveAcl">
          {{ t('nacos.save_acl') }}
        </a-button>
      </div>

      <a-alert
        v-if="aclSource"
        type="info"
        show-icon
        style="margin-bottom: 8px"
        :message="t('nacos.perm_load_source', { id: aclSource.id, name: aclSource.username })"
      />

      <div v-if="loading" style="text-align: center; padding: 16px"><a-spin /></div>

      <div style="margin-top: 16px">
        <a-alert
          v-if="permCatalog.length === 0"
          type="info"
          :message="t('nacos.perm_catalog_empty')"
        />
        <template v-else>
          <div style="display: flex; flex-wrap: wrap; align-items: center; gap: 8px 12px; margin-bottom: 12px">
            <span style="font-weight: 600; margin-right: 4px">{{ t('nacos.perm_rules_heading') }}</span>
            <a-button size="small" @click="selectAllPermissions">{{ t('nacos.perm_rules_select_all') }}</a-button>
            <a-button size="small" @click="clearAllPermissions">{{ t('nacos.perm_rules_clear_all') }}</a-button>
          </div>
          <div
            style="display: grid; grid-template-columns: repeat(auto-fill, minmax(min(100%, 260px), 1fr)); gap: 12px 16px; align-items: stretch"
          >
            <div
              v-for="item in permCatalog"
              :key="item.key"
              style="display: flex; align-items: flex-start; gap: 10px; padding: 10px 12px; border: 1px solid rgba(34,36,38,.12); border-radius: 6px; background: rgba(0,0,0,.02); min-width: 0"
            >
              <a-checkbox
                :checked="!!rules[item.key]"
                style="flex-shrink: 0; padding-top: 2px"
                @change="(e) => toggle(item.key, e.target.checked)"
              />
              <div style="cursor: pointer; flex: 1; min-width: 0" @click="toggleRule(item.key)">
                <div style="font-weight: 500; line-height: 1.35">
                  {{ useZh ? item.label_zh : item.label_en }}
                </div>
                <div
                  style="font-size: 0.85em; color: #767676; margin-top: 4px; word-break: break-word; overflow-wrap: anywhere"
                >
                  {{ item.key }}
                </div>
              </div>
            </div>
          </div>
        </template>
      </div>
    </a-card>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';
import { API, showError, showSuccess } from '@/helpers';

const { t, locale } = useI18n();

const roleLabel = (role) => {
  if (role >= 100) return 'Root';
  if (role >= 10) return 'Admin';
  return 'User';
};

const permCatalog = ref([]);
const rules = ref({});
const loading = ref(false);
const searching = ref(false);

const keyword = ref('');
const searchResults = ref([]);
const selectedIds = ref(new Set());
const aclSource = ref(null);

const userColumns = computed(() => [
  { title: t('nacos.perm_col_select'), key: 'select', align: 'center', width: 60 },
  { title: t('nacos.perm_col_id'), key: 'id' },
  { title: t('nacos.perm_col_username'), key: 'username' },
  { title: t('nacos.perm_col_display'), key: 'display' },
  { title: t('nacos.perm_col_email'), key: 'email' },
  { title: t('nacos.perm_col_role'), key: 'role' },
]);

const useZh = computed(
  () => locale.value && String(locale.value).toLowerCase().startsWith('zh')
);

const selectedCount = computed(() => selectedIds.value.size);

const normalizeCatalogItem = (raw) => {
  if (!raw || typeof raw !== 'object') return null;
  const key = raw.key ?? raw.Key ?? '';
  if (!key) return null;
  return {
    key,
    label_zh: raw.label_zh ?? raw.LabelZh ?? key,
    label_en: raw.label_en ?? raw.LabelEn ?? key,
  };
};

const loadInfo = async () => {
  const ir = await API.get('/api/nacos/registry/info');
  if (!ir.data?.success || !ir.data.data) {
    return;
  }
  const d = ir.data.data;
  if (Array.isArray(d.permission_catalog) && d.permission_catalog.length) {
    permCatalog.value = d.permission_catalog
      .map(normalizeCatalogItem)
      .filter(Boolean);
  } else if (Array.isArray(d.permission_keys)) {
    permCatalog.value = d.permission_keys.map((k) => ({
      key: k,
      label_zh: k,
      label_en: k,
    }));
  }
};

const searchUsers = async () => {
  const q = keyword.value.trim();
  if (q.length < 1) {
    showError(t('nacos.perm_search_need_keyword'));
    return;
  }
  searching.value = true;
  try {
    const res = await API.get('/api/nacos/users/search', { params: { keyword: q } });
    if (!res.data?.success) {
      showError(res.data?.message || 'search failed');
      return;
    }
    const list = Array.isArray(res.data.data) ? res.data.data : [];
    searchResults.value = list;
    selectedIds.value = new Set();
    aclSource.value = null;
    if (list.length === 0) {
      showError(t('nacos.perm_no_results'));
    }
  } catch (e) {
    showError(e.message);
  } finally {
    searching.value = false;
  }
};

const toggleUser = (id) => {
  const next = new Set(selectedIds.value);
  if (next.has(id)) {
    next.delete(id);
  } else {
    next.add(id);
  }
  selectedIds.value = next;
};

const selectAllInResults = () => {
  selectedIds.value = new Set(searchResults.value.map((u) => u.user_id));
};

const clearSelection = () => {
  selectedIds.value = new Set();
};

const loadAcl = async () => {
  if (selectedIds.value.size !== 1) {
    showError(t('nacos.perm_select_one_load'));
    return;
  }
  const id = [...selectedIds.value][0];
  loading.value = true;
  try {
    const res = await API.get(`/api/nacos/users/${id}/acl`);
    if (!res.data?.success) {
      showError(res.data?.message || 'load failed');
      return;
    }
    rules.value = res.data.data?.rules || {};
    const u = searchResults.value.find((x) => x.user_id === id);
    aclSource.value = { id, username: u?.username || String(id) };
  } catch (e) {
    showError(e.message);
  } finally {
    loading.value = false;
  }
};

const saveAcl = async () => {
  if (selectedIds.value.size < 1) {
    showError(t('nacos.perm_select_users_save'));
    return;
  }
  loading.value = true;
  const ids = [...selectedIds.value];
  let ok = 0;
  try {
    for (const id of ids) {
      const res = await API.put(`/api/nacos/users/${id}/acl`, { rules: rules.value });
      if (!res.data?.success) {
        showError(res.data?.message || `save failed for user #${id}`);
        return;
      }
      ok += 1;
    }
    showSuccess(t('nacos.perm_batch_saved', { count: ok }));
  } catch (e) {
    showError(e.message);
  } finally {
    loading.value = false;
  }
};

const toggle = (k, checked) => {
  rules.value = { ...rules.value, [k]: !!checked };
};

const toggleRule = (k) => {
  rules.value = { ...rules.value, [k]: !rules.value[k] };
};

const selectAllPermissions = () => {
  const next = { ...rules.value };
  permCatalog.value.forEach((item) => {
    next[item.key] = true;
  });
  rules.value = next;
};

const clearAllPermissions = () => {
  const next = { ...rules.value };
  permCatalog.value.forEach((item) => {
    next[item.key] = false;
  });
  rules.value = next;
};

onMounted(loadInfo);
</script>
