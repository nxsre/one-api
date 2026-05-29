# React → Vue migration guide (web/default → web/vue)

You are porting the One API frontend from **React 18 + Semantic UI React + react-router + i18next**
(`web/default/src`) to **Vue 3 `<script setup>` + Ant Design Vue 4 + vue-router + vue-i18n + Pinia + Tailwind**
(`web/vue/src`). Match behavior faithfully; do not redesign UX or change API calls.

## Source & target
- Source theme: `/opt/workspaces/uphone/k3d/xcph/one-api/web/default/src`
- Target theme: `/opt/workspaces/uphone/k3d/xcph/one-api/web/vue/src`
- Keep the SAME file/folder names, but `.js` → `.vue` (e.g. `pages/Token/index.js` → `pages/Token/index.vue`,
  `components/TokensTable.js` → `components/TokensTable.vue`).
- Overwrite the existing placeholder `.vue` stubs (read them first, they only contain an `<a-empty>`).

## Foundation already in place — REUSE, do not recreate
- API: `import { API } from '@/helpers'` (axios instance, interceptors, 401 handling all done).
- Helpers barrel: `import { ... } from '@/helpers'` — includes `showError/showSuccess/showInfo/showWarning/showNotice`,
  `renderQuota/renderQuotaWithPrompt/renderNumber/renderText/renderGroup/renderColorLabel/colorForLabel`,
  `copy, timestamp2string, downloadTextAsFile, verifyJSON, isAdmin, isRoot, isMobile, getChannelModels, loadChannelModels`,
  `isTenantAdmin, hasTenantPermission, postLoginDefaultPath`, captcha/secure-login helpers, `getOAuthState/onGitHubOAuthClicked/onLarkOAuthClicked`,
  model-catalog/operationRatioLookup/tokenModelDropdown/channelModelTest helpers (already copied verbatim).
- Stores: `import { useUserStore } from '@/stores/user'` and `import { useStatusStore } from '@/stores/status'`.
  User payload source of truth stays in `localStorage('user')`; use `userStore.login(payload)/logout()` to mutate.
- i18n: in `<script setup>` use `const { t } = useI18n()`; in template use `$t('...')` or `t('...')`.
  Locale keys are IDENTICAL to the React theme (`web/default/src/locales`), interpolation is `{name}` (single brace) here.
  Use `t('key', { name })` for interpolation.
- Router: routes already defined in `@/router`. Use `useRouter()/useRoute()`. Navigate with `router.push('/path')`,
  params via `route.params.id`. The React `useNavigate()` → `router.push`; `<Link to>` → `<router-link to>`.

## Component mapping (Semantic UI React → Ant Design Vue 4)
- `Table` → `a-table` (prefer `:columns` + `:data-source`; use `customRender`/slots for cells). Or keep simple `<a-table>` with `#bodyCell`.
- `Button` → `a-button` (`primary`→`type="primary"`, `negative`→`danger`, `loading`, `icon` via `#icon` slot + `@ant-design/icons-vue`).
- `Form/Form.Input/Form.Field` → `a-form` / `a-form-item` + `a-input` / `a-input-password` / `a-textarea`.
- `Dropdown` (select) → `a-select` with `:options` or `<a-select-option>`; `multiple` → `mode="multiple"`; `search` → `show-search`.
- `Modal` → `a-modal` (`v-model:open`). `Modal.Header/Content/Actions` → title prop / default slot / `#footer`.
- `Message` → `a-alert` (inline) or `message.*` (toast, via helpers showError/etc).
- `Label` → `a-tag`. `Icon name='x'` → an `@ant-design/icons-vue` component (pick the closest).
- `Pagination` → `a-pagination` or `a-table` built-in pagination.
- `Checkbox` → `a-checkbox`; `Popup` → `a-tooltip`/`a-popover`; `Segment`/`Card` → `a-card`.
- `Tab`/`Menu` (tabs) → `a-tabs`.
- Use Tailwind utility classes for layout/spacing instead of Semantic's `Grid`/`style` where convenient.

## React → Vue idioms
- `useState(x)` → `const x = ref(initial)`; object state → `reactive({...})`.
- `useEffect(fn, [])` → `onMounted(fn)`; `useEffect(fn, [dep])` → `watch(() => dep, fn)`.
- `useContext` → store (`useUserStore`/`useStatusStore`) or `useI18n`.
- props: `defineProps({...})`; emits: `defineEmits([...])`.
- Controlled inputs `value/onChange` → `v-model:value`.
- Conditional render `{cond && <X/>}` → `<X v-if="cond" />`; lists `arr.map(...)` → `v-for` with `:key`.

## Monaco editor
Some components (Settings, Routing JSON, Nacos config) use `@monaco-editor/react`. In Vue, use the
`monaco-editor` package directly. Create/reuse `@/components/MonacoEditor.vue` (a thin wrapper mounting
`monaco.editor.create` on a div, `v-model:value`, language/options props, dispose on unmount). Configure workers
via `self.MonacoEnvironment = { getWorker }` using `?worker` imports (e.g.
`import EditorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker'`). If a wrapper already exists, reuse it.

## Charts
`recharts` → `vue-echarts` + `echarts`. Import only needed echarts modules. Keep the same data transforms.

## Tables: prefer columns-config
For data tables, define `columns` with `title`/`dataIndex`/`key` and render cells via `#bodyCell="{ column, record }"`.
Keep sorting/filtering/search/pagination behavior equivalent to the source.

## Don'ts
- Don't change endpoint URLs, request/response handling, localStorage keys, or i18n keys.
- Don't add new dependencies without need (echarts/monaco-editor are allowed; both are or will be in package.json).
- Don't touch files outside your assigned scope.
- Don't leave `console.log` debugging or TODO stubs in migrated pages.

## After migrating your scope
- Ensure imports resolve and the SFC is syntactically valid Vue 3 `<script setup>`.
- Report exactly which files you created/overwrote.
