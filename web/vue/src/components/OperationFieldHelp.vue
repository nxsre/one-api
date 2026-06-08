<script>
import { defineComponent, h } from 'vue';
import { useI18n } from 'vue-i18n';
import { QuestionCircleOutlined } from '@ant-design/icons-vue';
import { Popover } from 'ant-design-vue';

function HelpSection(props, { slots }) {
  return h('div', { class: 'operation-billing-help-section' }, [
    props.title
      ? h('div', { class: 'operation-billing-help-section-title' }, props.title)
      : null,
    slots.default ? slots.default() : null,
  ]);
}
HelpSection.props = ['title'];

function HelpCode(_, { slots }) {
  return h('pre', { class: 'operation-billing-help-code' }, slots.default ? slots.default() : null);
}

export const BillingModeHelpContent = defineComponent({
  name: 'BillingModeHelpContent',
  setup() {
    const { t } = useI18n();
    return () =>
      h('div', {}, [
        h(HelpSection, { title: t('setting.operation.ratio.billing_mode.help.intro_title') }, () => [
          h('p', {}, t('setting.operation.ratio.billing_mode.help.intro')),
        ]),
        h(HelpSection, { title: t('setting.operation.ratio.billing_mode.help.values_title') }, () => [
          h('ul', {}, [
            h('li', {}, [
              h('strong', {}, 'ratio'),
              ` — ${t('setting.operation.ratio.billing_mode.help.ratio_desc')}`,
            ]),
            h('li', {}, [
              h('strong', {}, 'tiered_expr'),
              ` — ${t('setting.operation.ratio.billing_mode.help.tiered_desc')}`,
            ]),
          ]),
        ]),
        h(HelpSection, { title: t('setting.operation.ratio.billing_mode.help.example_title') }, () => [
          h(HelpCode, {}, () => `{
  "claude-sonnet-4-6": "tiered_expr",
  "gpt-4o": "ratio"
}`),
          h('p', {}, t('setting.operation.ratio.billing_mode.help.example_note')),
        ]),
      ]);
  },
});

export const BillingExprHelpContent = defineComponent({
  name: 'BillingExprHelpContent',
  setup() {
    const { t } = useI18n();
    return () =>
      h('div', {}, [
        h(HelpSection, { title: t('setting.operation.ratio.billing_expr.help.intro_title') }, () => [
          h('p', {}, t('setting.operation.ratio.billing_expr.help.intro')),
        ]),
        h(HelpSection, { title: t('setting.operation.ratio.billing_expr.help.vars_title') }, () => [
          h('table', { class: 'operation-billing-help-vars' }, [
            h('thead', {}, [
              h('tr', {}, [
                h('th', {}, t('setting.operation.ratio.billing_expr.help.var_col')),
                h('th', {}, t('setting.operation.ratio.billing_expr.help.meaning_col')),
              ]),
            ]),
            h('tbody', {}, [
              h('tr', {}, [
                h('td', {}, [h('code', {}, 'p'), ' / ', h('code', {}, 'c')]),
                h('td', {}, t('setting.operation.ratio.billing_expr.help.var_pc')),
              ]),
              h('tr', {}, [
                h('td', {}, [h('code', {}, 'len')]),
                h('td', {}, t('setting.operation.ratio.billing_expr.help.var_len')),
              ]),
              h('tr', {}, [
                h('td', {}, [
                  h('code', {}, 'cr'),
                  ' / ',
                  h('code', {}, 'cc'),
                  ' / ',
                  h('code', {}, 'cc1h'),
                ]),
                h('td', {}, t('setting.operation.ratio.billing_expr.help.var_cache')),
              ]),
              h('tr', {}, [
                h('td', {}, [
                  h('code', {}, 'img'),
                  ' / ',
                  h('code', {}, 'ai'),
                  ' / ',
                  h('code', {}, 'img_o'),
                  ' / ',
                  h('code', {}, 'ao'),
                ]),
                h('td', {}, t('setting.operation.ratio.billing_expr.help.var_media')),
              ]),
            ]),
          ]),
          h('p', {}, t('setting.operation.ratio.billing_expr.help.vars_note')),
        ]),
        h(HelpSection, { title: t('setting.operation.ratio.billing_expr.help.tier_title') }, () => [
          h('p', {}, t('setting.operation.ratio.billing_expr.help.tier_desc')),
          h(HelpCode, {}, () => 'tier("tier_name", p * 3 + c * 15 + cr * 0.3)'),
          h('p', {}, t('setting.operation.ratio.billing_expr.help.price_unit')),
        ]),
        h(HelpSection, { title: t('setting.operation.ratio.billing_expr.help.examples_title') }, () => [
          h('p', {}, t('setting.operation.ratio.billing_expr.help.ex1_label')),
          h(HelpCode, {}, () => `{
  "claude-sonnet-4-6": "tier(\\"base\\", p * 3 + c * 15 + cr * 0.3 + cc * 3.75 + cc1h * 6)"
}`),
          h('p', {}, t('setting.operation.ratio.billing_expr.help.ex2_label')),
          h(HelpCode, {}, () => `{
  "claude-sonnet-4-6": "len <= 200000 ? tier(\\"standard\\", p * 1.5 + c * 7.5) : tier(\\"long_context\\", p * 3.0 + c * 11.25)"
}`),
        ]),
        h(HelpSection, {}, () => [
          h('p', {}, t('setting.operation.ratio.billing_expr.help.relation')),
        ]),
      ]);
  },
});

export const FieldLabelWithHelp = defineComponent({
  name: 'FieldLabelWithHelp',
  props: {
    label: { type: String, default: '' },
    helpContent: { type: [Object, Function], default: null },
  },
  setup(props) {
    return () =>
      h('label', { class: 'operation-field-label-with-help' }, [
        h('span', {}, props.label),
        props.helpContent
          ? h(
              Popover,
              {
                placement: 'topLeft',
                overlayClassName: 'operation-billing-help-popup-wrap',
              },
              {
                content: () =>
                  h('div', { class: 'operation-billing-help-popup' }, [h(props.helpContent)]),
                default: () =>
                  h(
                    'span',
                    {
                      class: 'operation-field-help-trigger',
                      role: 'button',
                      tabindex: 0,
                      onClick: (e) => e.preventDefault(),
                    },
                    [
                      h(QuestionCircleOutlined, {
                        class: 'operation-field-help-icon',
                        'aria-label': 'help',
                      }),
                    ]
                  ),
              }
            )
          : null,
      ]);
  },
});

export default {
  name: 'OperationFieldHelp',
};
</script>

<style>
.operation-field-label-with-help {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  margin-bottom: 0.35rem;
}

.operation-field-help-trigger {
  display: inline-flex;
  align-items: center;
  line-height: 1;
}

.operation-field-help-icon {
  margin: 0 !important;
  color: rgba(0, 0, 0, 0.45) !important;
  cursor: help;
  font-size: 1em !important;
}

.operation-field-help-icon:hover {
  color: #2185d0 !important;
}

.operation-billing-help-popup-wrap {
  max-width: min(480px, 92vw) !important;
}

.operation-billing-help-popup {
  font-size: 13px;
  line-height: 1.55;
  color: rgba(0, 0, 0, 0.87);
  max-height: 70vh;
  overflow-y: auto;
}

.operation-billing-help-section + .operation-billing-help-section {
  margin-top: 0.75rem;
  padding-top: 0.75rem;
  border-top: 1px solid rgba(0, 0, 0, 0.08);
}

.operation-billing-help-section-title {
  font-weight: 600;
  margin-bottom: 0.35rem;
  color: rgba(0, 0, 0, 0.85);
}

.operation-billing-help-popup p {
  margin: 0 0 0.5rem;
}

.operation-billing-help-popup ul {
  margin: 0 0 0.5rem;
  padding-left: 1.25rem;
}

.operation-billing-help-popup li {
  margin-bottom: 0.25rem;
}

.operation-billing-help-code {
  margin: 0.35rem 0 0.5rem;
  padding: 0.5rem 0.65rem;
  background: rgba(0, 0, 0, 0.04);
  border-radius: 4px;
  font-size: 11px;
  line-height: 1.45;
  white-space: pre-wrap;
  word-break: break-word;
  overflow-x: auto;
}

.operation-billing-help-vars {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
  margin: 0.35rem 0 0.5rem;
}

.operation-billing-help-vars th,
.operation-billing-help-vars td {
  border: 1px solid rgba(0, 0, 0, 0.1);
  padding: 0.35rem 0.5rem;
  text-align: left;
  vertical-align: top;
}

.operation-billing-help-vars th {
  background: rgba(0, 0, 0, 0.04);
  font-weight: 600;
}

.operation-billing-help-vars code {
  font-size: 11px;
}

html.dark .operation-field-help-icon {
  color: rgba(255, 255, 255, 0.5) !important;
}

html.dark .operation-billing-help-popup {
  color: rgba(255, 255, 255, 0.9);
}

html.dark .operation-billing-help-section-title {
  color: rgba(255, 255, 255, 0.95);
}

html.dark .operation-billing-help-code,
html.dark .operation-billing-help-vars th {
  background: rgba(255, 255, 255, 0.08);
}

html.dark .operation-billing-help-vars th,
html.dark .operation-billing-help-vars td {
  border-color: rgba(255, 255, 255, 0.15);
}

html.dark .operation-billing-help-section + .operation-billing-help-section {
  border-top-color: rgba(255, 255, 255, 0.12);
}
</style>
