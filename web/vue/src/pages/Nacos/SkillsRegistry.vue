<template>
  <div class="dashboard-container">
    <a-card class="chart-card nacos-registry-table-card">
      <h3 class="header">{{ t('nacos.skills_title') }}</h3>
      <a-alert
        v-if="info"
        type="info"
        show-icon
        style="margin: 8px 0 12px"
        :message="infoText"
      />
      <div style="margin-bottom: 8px; color: #666; font-size: 13px">
        {{ t('nacos.skills_total', { total: totalCount }) }}
      </div>
      <div
        style="display: flex; flex-wrap: wrap; align-items: center; column-gap: 12px; row-gap: 8px; margin-bottom: 12px"
      >
        <span style="flex-shrink: 0; color: #666">namespace</span>
        <div style="flex: 1 1 220px; min-width: 180px; max-width: 420px">
          <NacosNamespaceSelect :value="namespace" @change="onNamespaceChange" />
        </div>
        <a-button size="small" style="margin: 0" @click="load">{{ t('nacos.refresh') }}</a-button>
        <a-button
          size="small"
          type="primary"
          style="margin: 0"
          @click="() => { uploadFile = null; uploadOpen = true; }"
        >
          {{ t('nacos.skills_upload') }}
        </a-button>
      </div>

      <!-- Filters -->
      <div
        class="nacos-skills-filter-row"
        style="display: flex; flex-wrap: wrap; align-items: flex-end; column-gap: 22px; row-gap: 14px; margin-bottom: 16px; padding-top: 2px"
      >
        <div style="min-width: 168px; flex: 0 1 auto">
          <label class="filter-label">{{ t('nacos.skills_search_name') }}</label>
          <a-input
            size="small"
            :placeholder="t('nacos.skills_search_placeholder')"
            v-model:value="searchInput"
            @keydown.enter="applyFilters"
          />
        </div>
        <div style="min-width: 168px; flex: 0 1 auto">
          <label class="filter-label">{{ t('nacos.skills_filter_biztag') }}</label>
          <a-input
            size="small"
            :placeholder="t('nacos.skills_filter_biztag_ph')"
            v-model:value="bizTagInput"
            @keydown.enter="applyFilters"
          />
        </div>
        <div v-if="rootUser" style="min-width: 168px; flex: 0 1 auto">
          <label class="filter-label">{{ t('nacos.skills_filter_owner') }}</label>
          <a-input
            size="small"
            :placeholder="t('nacos.skills_filter_owner_ph')"
            v-model:value="ownerInput"
            @keydown.enter="applyFilters"
          />
        </div>
        <a-button
          v-else
          size="small"
          :type="filterOnlyMine ? 'primary' : 'default'"
          style="align-self: flex-end; margin-bottom: 2px"
          @click="toggleFilterOnlyMine"
        >
          {{ t('nacos.skills_filter_only_mine') }}
        </a-button>
        <div style="min-width: 140px; flex: 0 1 auto">
          <label class="filter-label">{{ t('nacos.skills_filter_scope') }}</label>
          <a-select
            size="small"
            style="width: 100%"
            :value="filterScope || '_all'"
            :options="scopeOptions"
            @change="setScopeFilter"
          />
        </div>
        <div style="min-width: 168px; flex: 0 1 auto">
          <label class="filter-label">{{ t('nacos.skills_sort') }}</label>
          <a-select
            size="small"
            style="width: 100%"
            :value="orderBy || '_'"
            :options="sortOptions"
            @change="setSortOrder"
          />
        </div>
        <div style="display: flex; gap: 10px; align-items: center; flex-wrap: wrap; margin-left: 4px; align-self: flex-end; margin-bottom: 2px">
          <a-button size="small" type="primary" @click="applyFilters">{{ t('nacos.skills_search_btn') }}</a-button>
          <a-button size="small" @click="resetFilters">{{ t('nacos.skills_reset_filters') }}</a-button>
        </div>
        <div
          v-if="selectedNames.length > 0"
          style="margin-left: auto; display: flex; align-items: center; gap: 8px; flex-wrap: wrap"
        >
          <span style="font-size: 12px; color: #666">{{ t('nacos.skills_selected', { n: selectedNames.length }) }}</span>
          <a-button danger size="small" @click="batchDeleteOpen = true">{{ t('nacos.skills_batch_delete') }}</a-button>
          <a-button size="small" @click="clearSelection">{{ t('nacos.skills_clear_selection') }}</a-button>
        </div>
      </div>

      <div v-if="loading" style="text-align: center; padding: 24px"><a-spin /></div>
      <template v-else>
        <a-table
          size="small"
          class="nacos-skills-table"
          :columns="columns"
          :data-source="rows"
          :row-key="(r) => r.name"
          :pagination="false"
        >
          <template #headerCell="{ column }">
            <template v-if="column.key === 'select'">
              <a-checkbox
                :checked="allPageSelected"
                :indeterminate="someSelected"
                @change="toggleSelectAllPage"
              />
            </template>
          </template>
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'select'">
              <a-checkbox :checked="!!selected[record.name]" @change="() => toggleSelect(record.name)" />
            </template>
            <template v-else-if="column.key === 'name'">
              <span style="word-break: break-word">{{ record.name }}</span>
            </template>
            <template v-else-if="column.key === 'desc'">
              <span
                :title="record.description || ''"
                style="display: inline-block; max-width: 280px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; vertical-align: bottom"
              >{{ record.description || '—' }}</span>
            </template>
            <template v-else-if="column.key === 'owner'">{{ record.owner || '—' }}</template>
            <template v-else-if="column.key === 'enable'">
              <CheckCircleOutlined v-if="record.enable" style="color: #21ba45" />
              <span v-else style="color: #999">—</span>
            </template>
            <template v-else-if="column.key === 'scope'">{{ record.scope || 'PUBLIC' }}</template>
            <template v-else-if="column.key === 'draftReview'">
              <div style="font-size: 12px; line-height: 1.45">
                <div>
                  <span style="color: #888">{{ t('nacos.skills_col_draft_short') }}</span>
                  {{ record.editingVersion || '—' }}
                </div>
                <div>
                  <span style="color: #888">{{ t('nacos.skills_col_reviewing_short') }}</span>
                  {{ record.reviewingVersion || '—' }}
                </div>
              </div>
            </template>
            <template v-else-if="column.key === 'onlineCnt'">
              {{ record.onlineCnt != null ? record.onlineCnt : '—' }}
            </template>
            <template v-else-if="column.key === 'downloads'">
              {{ record.downloadCount != null ? record.downloadCount : '—' }}
            </template>
            <template v-else-if="column.key === 'actions'">
              <div style="display: flex; flex-wrap: wrap; gap: 6px; align-items: center">
                <a-button-group size="small">
                  <a-button @click="openDetail(record.name)">{{ t('nacos.skills_action_detail') }}</a-button>
                  <a-button @click="openEdit(record)">{{ t('nacos.skills_action_edit') }}</a-button>
                </a-button-group>
                <a-dropdown>
                  <a-button size="small">{{ t('nacos.skills_more_actions') }} <DownOutlined /></a-button>
                  <template #overlay>
                    <a-menu>
                      <a-menu-item @click="openSubmit(record)">{{ t('nacos.skills_action_submit') }}</a-menu-item>
                      <a-menu-item @click="openPublish(record.name, record.reviewingVersion)">{{ t('nacos.skills_action_publish') }}</a-menu-item>
                      <a-menu-item @click="openLabels(record)">{{ t('nacos.skills_action_labels') }}</a-menu-item>
                      <a-menu-divider />
                      <a-menu-item @click="() => { deleteName = record.name; deleteOpen = true; }">
                        <span style="color: #db2828">{{ t('nacos.skills_action_delete') }}</span>
                      </a-menu-item>
                    </a-menu>
                  </template>
                </a-dropdown>
              </div>
            </template>
          </template>
        </a-table>

        <div
          v-if="totalCount > pageSize"
          style="display: flex; align-items: center; justify-content: flex-end; gap: 8px; margin-top: 12px; flex-wrap: wrap"
        >
          <span style="font-size: 13px; color: #666">
            {{ t('nacos.skills_page_summary', { pageNo, totalPages, total: totalCount }) }}
          </span>
          <a-select
            size="small"
            style="width: 120px"
            :value="pageSize"
            :options="pageSizeOptions"
            @change="onPageSizeChange"
          />
          <a-button size="small" :disabled="pageNo <= 1" @click="pageNo = Math.max(1, pageNo - 1)">
            {{ t('nacos.skills_prev_page') }}
          </a-button>
          <a-button size="small" :disabled="pageNo >= totalPages" @click="pageNo = Math.min(totalPages, pageNo + 1)">
            {{ t('nacos.skills_next_page') }}
          </a-button>
        </div>
      </template>
    </a-card>

    <!-- Detail -->
    <a-modal
      v-model:open="detailOpen"
      :title="t('nacos.skills_detail_title')"
      width="80%"
      :footer="null"
      @cancel="detailOpen = false"
    >
      <div v-if="detailLoading" style="text-align: center; padding: 24px"><a-spin /></div>
      <div v-else-if="detail">
        <p><strong>{{ detail.name }}</strong> — {{ detail.description }}</p>
        <p>
          {{ t('nacos.skills_detail_owner') }}: {{ detail.owner || '—' }} |
          {{ t('nacos.skills_col_enable') }}:
          {{ detail.enable ? t('nacos.skills_yes') : t('nacos.skills_no') }}
          | {{ t('nacos.skills_col_scope') }}: {{ detail.scope || 'PUBLIC' }} |
          bizTags: {{ detail.bizTags || '—' }}
        </p>
        <p v-if="detail.labels && Object.keys(detail.labels).length > 0">
          labels: {{ JSON.stringify(detail.labels) }}
        </p>
        <a-table
          size="small"
          :columns="versionColumns"
          :data-source="detail.versions || []"
          :row-key="(v) => v.version"
          :pagination="false"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'version'">{{ record.version }}</template>
            <template v-else-if="column.key === 'status'">{{ skillVersionStatusLabel(record.status) }}</template>
            <template v-else-if="column.key === 'commit'">{{ record.commitMsg || '—' }}</template>
            <template v-else-if="column.key === 'actions'">
              <div style="display: flex; flex-wrap: wrap; gap: 6px; align-items: center">
                <a-button
                  v-if="record.status === 'online'"
                  size="small"
                  @click="downloadVersionZip(detail.name, record.version)"
                >
                  {{ t('nacos.skills_download') }}
                </a-button>
                <span v-else style="color: #999; font-size: 12px">{{ t('nacos.skills_download_online_only') }}</span>
                <a-button
                  v-if="record.status === 'online'"
                  size="small"
                  :loading="isVersionOpBusy(detail.name, record.version, 'off')"
                  :disabled="!!versionOpBusy && !isVersionOpBusy(detail.name, record.version, 'off')"
                  @click="postSkillVersionOffline(detail.name, record.version)"
                >
                  {{ t('nacos.skills_action_offline') }}
                </a-button>
                <a-button
                  v-if="record.status === 'offline'"
                  type="primary"
                  size="small"
                  :loading="isVersionOpBusy(detail.name, record.version, 'on')"
                  :disabled="!!versionOpBusy && !isVersionOpBusy(detail.name, record.version, 'on')"
                  @click="postSkillVersionOnline(detail.name, record.version)"
                >
                  {{ t('nacos.skills_action_online') }}
                </a-button>
              </div>
            </template>
          </template>
        </a-table>
      </div>
      <template #footer>
        <a-button @click="detailOpen = false">{{ t('nacos.skills_close') }}</a-button>
      </template>
    </a-modal>

    <!-- Edit -->
    <a-modal
      v-model:open="editOpen"
      :title="`${t('nacos.skills_edit_title')}: ${editName}`"
      :ok-text="t('nacos.skills_save')"
      :cancel-text="t('nacos.skills_close')"
      @ok="saveMetadata"
      @cancel="editOpen = false"
    >
      <a-form layout="vertical">
        <a-form-item :label="t('nacos.skills_col_desc')">
          <a-textarea
            v-model:value="editDesc"
            :auto-size="{ minRows: 3, maxRows: 8 }"
            allow-clear
          />
        </a-form-item>
        <a-row :gutter="16">
          <a-col :span="16">
            <a-form-item label="bizTags">
              <a-input v-model:value="editBiz" allow-clear />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item :label="t('nacos.skills_col_scope')">
              <a-select v-model:value="editScope" style="width: 100%">
                <a-select-option value="PUBLIC">PUBLIC</a-select-option>
                <a-select-option value="PRIVATE">PRIVATE</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item>
          <a-checkbox v-model:checked="editEnable">{{ t('nacos.skills_col_enable') }}</a-checkbox>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- Upload -->
    <a-modal
      v-model:open="uploadOpen"
      :title="t('nacos.skills_upload_title')"
      :ok-text="t('nacos.skills_upload')"
      :cancel-text="t('nacos.skills_close')"
      @ok="doUpload"
      @cancel="uploadOpen = false"
    >
      <p>{{ t('nacos.skills_upload_hint') }}</p>
      <input type="file" accept=".zip,application/zip" @change="onFileChange" />
    </a-modal>

    <!-- Submit -->
    <a-modal
      v-model:open="submitOpen"
      :title="`${t('nacos.skills_submit_title')}: ${submitName}`"
      :ok-text="t('nacos.skills_action_submit')"
      :cancel-text="t('nacos.skills_close')"
      @ok="doSubmit"
      @cancel="submitOpen = false"
    >
      <a-form layout="vertical">
        <SettingMonacoField
          :label="t('nacos.skills_version_optional')"
          :hint="t('nacos.skills_submit_hint')"
          :value="submitVersion"
          :origin-value="submitVersionBaseline"
          :height="88"
          @update:value="(v) => (submitVersion = v)"
        />
      </a-form>
    </a-modal>

    <!-- Publish -->
    <a-modal
      v-model:open="publishOpen"
      :title="`${t('nacos.skills_publish_title')}: ${publishName}`"
      :ok-text="t('nacos.skills_action_publish')"
      :cancel-text="t('nacos.skills_close')"
      :ok-button-props="{ type: 'primary' }"
      @ok="doPublish"
      @cancel="publishOpen = false"
    >
      <a-form layout="vertical">
        <SettingMonacoField
          :label="t('nacos.skills_publish_version')"
          :hint="`${t('nacos.skills_publish_hint')} · ${publishCandidates.length ? publishCandidates.join(', ') : '—'}`"
          :value="publishVersion"
          :origin-value="publishVersionBaseline"
          :height="96"
          @update:value="(v) => (publishVersion = v)"
        />
        <a-form-item>
          <a-checkbox v-model:checked="publishUpdateLatest">{{ t('nacos.skills_update_latest') }}</a-checkbox>
        </a-form-item>
        <a-form-item>
          <a-checkbox v-model:checked="publishForce">{{ t('nacos.skills_force_publish') }}</a-checkbox>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- Labels -->
    <a-modal
      v-model:open="labelsOpen"
      :title="`${t('nacos.skills_labels_title')}: ${labelsName}`"
      width="80%"
      :ok-text="t('nacos.skills_save')"
      :cancel-text="t('nacos.skills_close')"
      @ok="saveLabels"
      @cancel="labelsOpen = false"
    >
      <a-form layout="vertical">
        <SettingMonacoField
          label="labels (JSON object)"
          language="json"
          enable-json-format
          minimap
          :value="labelsText"
          :origin-value="labelsBaseline"
          :height="360"
          @update:value="(v) => (labelsText = v)"
        />
        <a-form-item>
          <a-checkbox v-model:checked="labelsReplace">{{ t('nacos.skills_labels_replace') }}</a-checkbox>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- Batch delete -->
    <a-modal
      v-model:open="batchDeleteOpen"
      :title="t('nacos.skills_batch_delete')"
      :ok-button-props="{ danger: true }"
      @ok="doBatchDelete"
      @cancel="batchDeleteOpen = false"
    >
      <p>{{ t('nacos.skills_batch_delete_confirm', { n: selectedNames.length }) }}</p>
    </a-modal>

    <!-- Delete -->
    <a-modal
      v-model:open="deleteOpen"
      :title="t('nacos.skills_delete_confirm')"
      :ok-button-props="{ danger: true }"
      @ok="doDelete"
      @cancel="() => { deleteOpen = false; deleteName = ''; }"
    >
      <p>{{ `${deleteName} @ ${namespace}` }}</p>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { CheckCircleOutlined, DownOutlined } from '@ant-design/icons-vue';
import {
  API,
  getStoredNacosNamespace,
  setStoredNacosNamespace,
  isRoot,
  showError,
  showSuccess,
} from '@/helpers';
import NacosNamespaceSelect from '@/components/NacosNamespaceSelect.vue';
import SettingMonacoField from '@/components/SettingMonacoField.vue';

const { t } = useI18n();

const fmtBytes = (n) => {
  if (n == null || !Number.isFinite(Number(n))) return '';
  const v = Number(n);
  if (v < 1024) return `${v} B`;
  if (v < 1024 * 1024) return `${(v / 1024).toFixed(1)} KiB`;
  return `${(v / (1024 * 1024)).toFixed(1)} MiB`;
};

const localUser = () => {
  try {
    const raw = localStorage.getItem('user');
    return raw ? JSON.parse(raw) : null;
  } catch {
    return null;
  }
};

const rootUser = computed(() => isRoot());

const loading = ref(true);
const info = ref(null);
const rows = ref([]);
const totalCount = ref(0);
const pageNo = ref(1);
const pageSize = ref(12);
const namespace = ref(getStoredNacosNamespace());

const searchInput = ref('');
const bizTagInput = ref('');
const ownerInput = ref('');
const filterScope = ref('');
const orderBy = ref('');
const filterOnlyMine = ref(false);
const query = ref({ name: '', biz: '', scope: '', order: '', owner: '', onlyMine: false });
const selected = ref({});
const batchDeleteOpen = ref(false);

const detailOpen = ref(false);
const detailLoading = ref(false);
const detail = ref(null);

const editOpen = ref(false);
const editName = ref('');
const editDesc = ref('');
const editBiz = ref('');
const editScope = ref('PUBLIC');
const editEnable = ref(true);
const editBaseline = ref({ desc: '', biz: '', scope: 'PUBLIC' });

const uploadOpen = ref(false);
const uploadFile = ref(null);

const deleteOpen = ref(false);
const deleteName = ref('');

const submitOpen = ref(false);
const submitName = ref('');
const submitVersion = ref('');
const submitVersionBaseline = ref('');

const publishOpen = ref(false);
const publishName = ref('');
const publishVersion = ref('');
const publishVersionBaseline = ref('');
const publishUpdateLatest = ref(true);
const publishForce = ref(false);
const publishCandidates = ref([]);

const labelsOpen = ref(false);
const labelsName = ref('');
const labelsText = ref('{}');
const labelsReplace = ref(false);
const labelsBaseline = ref('{}');

const versionOpBusy = ref(null);

const scopeOptions = computed(() => [
  { value: '_all', label: t('nacos.skills_filter_scope_all') },
  { value: 'PUBLIC', label: 'PUBLIC' },
  { value: 'PRIVATE', label: 'PRIVATE' },
]);

const sortOptions = computed(() => [
  { value: '_', label: t('nacos.skills_sort_default') },
  { value: 'download_count', label: t('nacos.skills_sort_downloads') },
]);

const pageSizeOptions = computed(() => [
  { value: 12, label: `12 / ${t('nacos.skills_page')}` },
  { value: 24, label: `24 / ${t('nacos.skills_page')}` },
  { value: 48, label: `48 / ${t('nacos.skills_page')}` },
]);

const columns = computed(() => [
  { title: '', key: 'select', width: 40 },
  { title: t('nacos.skills_col_name'), key: 'name' },
  { title: t('nacos.skills_col_desc'), key: 'desc' },
  { title: t('nacos.skills_col_owner'), key: 'owner' },
  { title: t('nacos.skills_col_enable'), key: 'enable', align: 'center' },
  { title: t('nacos.skills_col_scope'), key: 'scope' },
  { title: t('nacos.skills_col_draft_review'), key: 'draftReview' },
  { title: t('nacos.skills_col_online_cnt'), key: 'onlineCnt' },
  { title: t('nacos.skills_col_downloads'), key: 'downloads' },
  { title: t('nacos.skills_col_actions'), key: 'actions' },
]);

const versionColumns = computed(() => [
  { title: t('nacos.skills_col_version'), key: 'version' },
  { title: t('nacos.skills_col_status'), key: 'status' },
  { title: t('nacos.skills_col_commit'), key: 'commit' },
  { title: t('nacos.skills_col_actions'), key: 'actions' },
]);

const infoText = computed(() => {
  const i = info.value;
  if (!i) return '';
  let s = `${t('nacos.registry_storage')}: ${i.zip_storage}`;
  if (i.zip_local_dir) s += ` | 本地: ${i.zip_local_dir}`;
  if (i.s3_remote_configured) s += ' | S3 已配置';
  if (i.max_upload_bytes)
    s += ` | ${t('nacos.skills_max_upload')}: ${fmtBytes(i.max_upload_bytes)}`;
  return s;
});

const totalPages = computed(() =>
  Math.max(1, Math.ceil(totalCount.value / pageSize.value) || 1)
);

const selectedNames = computed(() =>
  Object.keys(selected.value).filter((k) => selected.value[k])
);

const allPageSelected = computed(
  () => rows.value.length > 0 && rows.value.every((r) => selected.value[r.name])
);

const someSelected = computed(
  () =>
    selectedNames.value.length > 0 &&
    !rows.value.every((r) => selected.value[r.name])
);

const skillVersionStatusLabel = (status) => {
  switch (status) {
    case 'draft':
      return t('nacos.skills_ver_draft');
    case 'reviewing':
      return t('nacos.skills_ver_reviewing');
    case 'reviewed':
      return t('nacos.skills_ver_reviewed');
    case 'online':
      return t('nacos.skills_ver_online');
    case 'offline':
      return t('nacos.skills_ver_offline');
    default:
      return status || '—';
  }
};

const applyFilters = () => {
  const u = localUser();
  query.value = {
    name: searchInput.value.trim(),
    biz: bizTagInput.value.trim(),
    scope: filterScope.value,
    order: orderBy.value,
    owner: isRoot() ? ownerInput.value.trim() : '',
    onlyMine: !isRoot() && filterOnlyMine.value && !!u?.username,
  };
  pageNo.value = 1;
};

const resetFilters = () => {
  searchInput.value = '';
  bizTagInput.value = '';
  ownerInput.value = '';
  filterScope.value = '';
  orderBy.value = '';
  filterOnlyMine.value = false;
  query.value = { name: '', biz: '', scope: '', order: '', owner: '', onlyMine: false };
  pageNo.value = 1;
};

const setScopeFilter = (v) => {
  const scope = !v || v === '_all' ? '' : v;
  filterScope.value = scope;
  query.value = { ...query.value, scope };
  pageNo.value = 1;
};

const setSortOrder = (v) => {
  const order = !v || v === '_' ? '' : v;
  orderBy.value = order;
  query.value = { ...query.value, order };
  pageNo.value = 1;
};

const toggleFilterOnlyMine = () => {
  const u = localUser();
  const next = !filterOnlyMine.value;
  filterOnlyMine.value = next;
  query.value = { ...query.value, onlyMine: next && !isRoot() && !!u?.username };
  pageNo.value = 1;
};

const toggleSelect = (name) => {
  selected.value = { ...selected.value, [name]: !selected.value[name] };
};

const toggleSelectAllPage = () => {
  const names = rows.value.map((r) => r.name);
  const allOn = names.length > 0 && names.every((n) => selected.value[n]);
  const next = { ...selected.value };
  if (allOn) {
    names.forEach((n) => delete next[n]);
  } else {
    names.forEach((n) => (next[n] = true));
  }
  selected.value = next;
};

const clearSelection = () => {
  selected.value = {};
};

const onPageSizeChange = (v) => {
  pageSize.value = Number(v);
  pageNo.value = 1;
};

const onFileChange = (e) => {
  uploadFile.value = e.target.files && e.target.files[0];
};

const load = async () => {
  loading.value = true;
  try {
    const ir = await API.get('/api/nacos/registry/info');
    if (!ir.data?.success) {
      showError(ir.data?.message || 'load info failed');
      return;
    }
    info.value = ir.data.data;
    const u = localUser();
    const params = {
      namespace: namespace.value,
      page: pageNo.value,
      size: pageSize.value,
    };
    if (query.value.name) {
      params.skillName = query.value.name;
      params.search = 'blur';
    }
    if (query.value.biz) params.bizTag = query.value.biz;
    if (query.value.scope) params.scope = query.value.scope;
    if (query.value.order) params.orderBy = query.value.order;
    if (isRoot() && query.value.owner) {
      params.owner = query.value.owner;
    } else if (query.value.onlyMine && u?.username) {
      params.owner = u.username;
    }
    const sr = await API.get('/api/nacos/skills', { params });
    if (!sr.data?.success) {
      showError(sr.data?.message || 'load skills failed');
      return;
    }
    rows.value = sr.data.data?.pageItems || [];
    totalCount.value = sr.data.data?.totalCount ?? 0;
    const nameSet = new Set(rows.value.map((r) => r.name));
    const next = { ...selected.value };
    Object.keys(next).forEach((k) => {
      if (!nameSet.has(k)) delete next[k];
    });
    selected.value = next;
  } catch (e) {
    showError(e.message);
  } finally {
    loading.value = false;
  }
};

watch(namespace, () => {
  setStoredNacosNamespace(namespace.value);
  pageNo.value = 1;
});

watch(
  [namespace, pageNo, pageSize, query],
  () => {
    load();
  },
  { immediate: true, deep: true }
);

const openDetail = async (name) => {
  detailOpen.value = true;
  detailLoading.value = true;
  detail.value = null;
  try {
    const res = await API.get('/api/nacos/skills/detail', {
      params: { namespace: namespace.value, name },
    });
    if (!res.data?.success) {
      showError(res.data?.message || 'load detail failed');
      detailOpen.value = false;
      return;
    }
    detail.value = res.data.data;
  } catch (e) {
    showError(e.message);
    detailOpen.value = false;
  } finally {
    detailLoading.value = false;
  }
};

const openEdit = (r) => {
  editName.value = r.name;
  const desc = r.description || '';
  const biz = r.bizTags || '';
  const scope = r.scope || 'PUBLIC';
  editDesc.value = desc;
  editBiz.value = biz;
  editScope.value = scope;
  editEnable.value = !!r.enable;
  editBaseline.value = { desc, biz, scope };
  editOpen.value = true;
};

const openSubmit = (r) => {
  submitName.value = r.name;
  submitVersion.value = '';
  submitVersionBaseline.value = '';
  submitOpen.value = true;
};

const openLabels = (r) => {
  const lt = JSON.stringify(r.labels || {}, null, 2);
  labelsName.value = r.name;
  labelsText.value = lt;
  labelsBaseline.value = lt;
  labelsReplace.value = false;
  labelsOpen.value = true;
};

const openPublish = async (name, reviewingFromList) => {
  publishName.value = name;
  publishUpdateLatest.value = true;
  publishForce.value = false;
  let candidates = [];
  if (reviewingFromList) {
    candidates = [reviewingFromList];
  }
  try {
    const res = await API.get('/api/nacos/skills/detail', {
      params: { namespace: namespace.value, name },
    });
    if (res.data?.success) {
      const rev = (res.data.data?.versions || [])
        .filter((v) => v.status === 'reviewing')
        .map((v) => v.version);
      if (rev.length) {
        candidates = rev;
      }
    }
  } catch {
    /* ignore */
  }
  publishCandidates.value = candidates;
  const initial = candidates[0] || '';
  publishVersion.value = initial;
  publishVersionBaseline.value = initial;
  publishOpen.value = true;
};

const isVersionOpBusy = (skillName, version, op) =>
  versionOpBusy.value?.name === skillName &&
  versionOpBusy.value?.version === version &&
  versionOpBusy.value?.op === op;

const postSkillVersionOnline = async (skillName, version) => {
  versionOpBusy.value = { name: skillName, version, op: 'on' };
  try {
    const res = await API.post('/api/nacos/skills/version/online', {
      namespace: namespace.value,
      name: skillName,
      version,
      updateLatestLabel: true,
    });
    if (!res.data?.success) {
      showError(res.data?.message || 'online failed');
      return;
    }
    showSuccess(t('nacos.skills_version_online_ok'));
    load();
    if (detailOpen.value && detail.value?.name === skillName) {
      openDetail(skillName);
    }
  } catch (e) {
    showError(e.message);
  } finally {
    versionOpBusy.value = null;
  }
};

const postSkillVersionOffline = async (skillName, version) => {
  versionOpBusy.value = { name: skillName, version, op: 'off' };
  try {
    const res = await API.post('/api/nacos/skills/version/offline', {
      namespace: namespace.value,
      name: skillName,
      version,
    });
    if (!res.data?.success) {
      showError(res.data?.message || 'offline failed');
      return;
    }
    showSuccess(t('nacos.skills_version_offline_ok'));
    load();
    if (detailOpen.value && detail.value?.name === skillName) {
      openDetail(skillName);
    }
  } catch (e) {
    showError(e.message);
  } finally {
    versionOpBusy.value = null;
  }
};

const saveMetadata = async () => {
  try {
    const res = await API.put('/api/nacos/skills/metadata', {
      namespace: namespace.value,
      name: editName.value,
      description: editDesc.value,
      bizTags: editBiz.value,
      scope: editScope.value,
      enable: editEnable.value,
    });
    if (!res.data?.success) {
      showError(res.data?.message || 'save failed');
      return;
    }
    showSuccess(t('nacos.skills_saved'));
    editOpen.value = false;
    load();
  } catch (e) {
    showError(e.message);
  }
};

const doUpload = async () => {
  if (!uploadFile.value) {
    showError(t('nacos.skills_pick_file'));
    return;
  }
  const fd = new FormData();
  fd.append('file', uploadFile.value);
  try {
    const res = await API.post(
      `/api/nacos/skills/upload?namespace=${encodeURIComponent(namespace.value)}`,
      fd
    );
    if (!res.data?.success) {
      showError(res.data?.message || 'upload failed');
      return;
    }
    showSuccess(t('nacos.skills_upload_ok'));
    uploadOpen.value = false;
    uploadFile.value = null;
    load();
  } catch (e) {
    showError(e.message);
  }
};

const doDelete = async () => {
  try {
    const res = await API.delete('/api/nacos/skills/item', {
      params: { namespace: namespace.value, name: deleteName.value },
    });
    if (!res.data?.success) {
      showError(res.data?.message || 'delete failed');
      return;
    }
    showSuccess(t('nacos.skills_deleted'));
    deleteOpen.value = false;
    deleteName.value = '';
    clearSelection();
    load();
  } catch (e) {
    showError(e.message);
  }
};

const doBatchDelete = async () => {
  const names = selectedNames.value;
  if (names.length === 0) return;
  try {
    for (const name of names) {
      const res = await API.delete('/api/nacos/skills/item', {
        params: { namespace: namespace.value, name },
      });
      if (!res.data?.success) {
        showError(res.data?.message || `${name}`);
        return;
      }
    }
    showSuccess(t('nacos.skills_batch_deleted'));
    clearSelection();
    batchDeleteOpen.value = false;
    load();
  } catch (e) {
    showError(e.message);
  }
};

const doSubmit = async () => {
  try {
    const res = await API.post('/api/nacos/skills/submit', {
      namespace: namespace.value,
      name: submitName.value,
      version: submitVersion.value || undefined,
    });
    if (!res.data?.success) {
      showError(res.data?.message || 'submit failed');
      return;
    }
    showSuccess(t('nacos.skills_submit_ok'));
    submitOpen.value = false;
    submitVersion.value = '';
    submitVersionBaseline.value = '';
    load();
    if (detailOpen.value && detail.value?.name === submitName.value) {
      openDetail(submitName.value);
    }
  } catch (e) {
    showError(e.message);
  }
};

const doPublish = async () => {
  if (!publishVersion.value.trim()) {
    showError(t('nacos.skills_version_required'));
    return;
  }
  try {
    const res = await API.post('/api/nacos/skills/publish', {
      namespace: namespace.value,
      name: publishName.value,
      version: publishVersion.value.trim(),
      updateLatest: publishUpdateLatest.value,
      forcePublish: publishForce.value,
    });
    if (!res.data?.success) {
      showError(res.data?.message || 'publish failed');
      return;
    }
    showSuccess(t('nacos.skills_publish_ok'));
    publishOpen.value = false;
    publishVersion.value = '';
    load();
    if (detailOpen.value && detail.value?.name === publishName.value) {
      openDetail(publishName.value);
    }
  } catch (e) {
    showError(e.message);
  }
};

const downloadVersionZip = async (name, ver) => {
  try {
    const res = await API.get('/api/nacos/skills/download', {
      params: { namespace: namespace.value, name, version: ver },
      responseType: 'blob',
    });
    const blob = new Blob([res.data], { type: 'application/zip' });
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${name}-${ver}.zip`;
    a.click();
    window.URL.revokeObjectURL(url);
  } catch (e) {
    showError(e.message);
  }
};

const saveLabels = async () => {
  let labels;
  try {
    labels = JSON.parse(labelsText.value || '{}');
  } catch {
    showError(t('nacos.skills_labels_invalid'));
    return;
  }
  try {
    const res = await API.post('/api/nacos/skills/labels', {
      namespace: namespace.value,
      name: labelsName.value,
      labels,
      replace: labelsReplace.value,
    });
    if (!res.data?.success) {
      showError(res.data?.message || 'save labels failed');
      return;
    }
    showSuccess(t('nacos.skills_labels_saved'));
    labelsOpen.value = false;
    load();
    if (detailOpen.value && detail.value?.name === labelsName.value) {
      openDetail(labelsName.value);
    }
  } catch (e) {
    showError(e.message);
  }
};
</script>

<style scoped>
.filter-label {
  font-size: 12px;
  color: #666;
  display: block;
  margin-bottom: 8px;
  font-weight: 500;
}
</style>
