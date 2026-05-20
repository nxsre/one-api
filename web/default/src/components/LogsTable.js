import React, { useEffect, useState } from 'react';
import {
  Button,
  Form,
  Header,
  Label,
  Loader,
  Modal,
  Pagination,
  Segment,
  Select,
  Table,
} from 'semantic-ui-react';
import {
  API,
  copy,
  isAdmin,
  showError,
  showSuccess,
  showWarning,
  timestamp2string,
  noAutofillTextProps,
} from '../helpers';
import { useTranslation } from 'react-i18next';

import { ITEMS_PER_PAGE } from '../constants';
import { renderColorLabel, renderQuota } from '../helpers/render';
import { Link } from 'react-router-dom';

function renderTimestamp(timestamp, request_id) {
  const full = timestamp2string(timestamp);
  return (
    <span
      className='log-time-cell'
      title={request_id ? `${full}\n请求 ID：${request_id}（点击复制）` : full}
      onClick={async () => {
        if (!request_id) return;
        if (await copy(request_id)) {
          showSuccess(`已复制请求 ID：${request_id}`);
        } else {
          showWarning(`请求 ID 复制失败：${request_id}`);
        }
      }}
      role={request_id ? 'button' : undefined}
      tabIndex={request_id ? 0 : undefined}
    >
      {full}
    </span>
  );
}

const MODE_OPTIONS = [
  { key: 'all', text: '全部用户', value: 'all' },
  { key: 'self', text: '当前用户', value: 'self' },
];

function renderType(type) {
  switch (type) {
    case 1:
      return (
        <Label basic color='green'>
          充值
        </Label>
      );
    case 2:
      return (
        <Label basic color='olive'>
          消费
        </Label>
      );
    case 3:
      return (
        <Label basic color='orange'>
          管理
        </Label>
      );
    case 4:
      return (
        <Label basic color='purple'>
          系统
        </Label>
      );
    case 5:
      return (
        <Label basic color='violet'>
          测试
        </Label>
      );
    case 6:
      return (
        <Label basic color='red'>
          错误
        </Label>
      );
    case 7:
      return (
        <Label basic color='teal'>
          退款
        </Label>
      );
    default:
      return (
        <Label basic color='black'>
          未知
        </Label>
      );
  }
}

function parseLogOther(other) {
  const s = other != null ? String(other).trim() : '';
  if (!s) {
    return null;
  }
  try {
    const o = JSON.parse(s);
    if (o && typeof o === 'object' && !Array.isArray(o)) {
      return o;
    }
  } catch {
    /* 非 JSON，按纯文本展示 */
  }
  return null;
}

function userInputLabel(t) {
  const key = t('log.table.user_input');
  return key !== 'log.table.user_input' ? key : '用户输入';
}

const LOG_OTHER_KNOWN_KEYS = new Set([
  'user_input',
  'http_status',
  'http_method',
  'http_path',
  'client_request_headers',
  'client_response_headers',
  'upstream_request_headers',
  'upstream_response_headers',
  'upstream_request_meta',
  'upstream_response_meta',
  'client_ip',
  'xff',
  'request_body_bytes',
  'response_body_bytes',
  'system_prompt_reset',
]);

function detailLabel(t, key, fallback) {
  const v = t(key);
  return v !== key ? v : fallback;
}

function appendDetailSection(parts, title, body) {
  const text = body != null ? String(body).trim() : '';
  if (!text) return;
  parts.push(`${title}\n${text}`);
}

function normalizeHeaderMap(headers) {
  if (!headers || typeof headers !== 'object') return {};
  const out = {};
  Object.entries(headers).forEach(([k, v]) => {
    if (v == null) return;
    out[k] = Array.isArray(v) ? v.map(String) : [String(v)];
  });
  return out;
}

/** curl -v 风格：> 请求头 / < 响应头 */
function formatCurlHeaders(prefix, headers, statusLine) {
  const lines = [];
  if (statusLine) {
    lines.push(`${prefix} ${statusLine}`);
  }
  Object.keys(headers)
    .sort((a, b) => a.localeCompare(b))
    .forEach((k) => {
      headers[k].forEach((v) => {
        lines.push(`${prefix} ${k}: ${v}`);
      });
    });
  lines.push(prefix);
  return lines.join('\n');
}

function upstreamRequestPath(meta) {
  const url = meta?.url != null ? String(meta.url) : '';
  if (!url) return '/';
  try {
    const u = new URL(url);
    return u.pathname + (u.search || '') || '/';
  } catch {
    return url;
  }
}

function buildClientRequestCurl(parsed) {
  if (!parsed?.client_request_headers) return '';
  const method = parsed.http_method || 'GET';
  const path = parsed.http_path || '/';
  return formatCurlHeaders(
    '>',
    normalizeHeaderMap(parsed.client_request_headers),
    `${method} ${path} HTTP/1.1`,
  );
}

function buildClientResponseCurl(parsed) {
  if (!parsed?.client_response_headers) return '';
  const code = parsed.http_status;
  const statusLine =
    code != null && code !== '' ? `HTTP/1.1 ${code}` : 'HTTP/1.1';
  return formatCurlHeaders(
    '<',
    normalizeHeaderMap(parsed.client_response_headers),
    statusLine,
  );
}

function buildUpstreamRequestCurl(parsed) {
  if (!parsed?.upstream_request_headers) return '';
  const meta = parsed.upstream_request_meta || {};
  const method = meta.method || 'GET';
  const path = upstreamRequestPath(meta);
  return formatCurlHeaders(
    '>',
    normalizeHeaderMap(parsed.upstream_request_headers),
    `${method} ${path} HTTP/1.1`,
  );
}

function buildUpstreamResponseCurl(parsed) {
  if (!parsed?.upstream_response_headers) return '';
  const meta = parsed.upstream_response_meta || {};
  let statusLine = 'HTTP/1.1';
  if (meta.status) {
    statusLine = String(meta.status);
    if (!statusLine.startsWith('HTTP')) {
      statusLine = `HTTP/1.1 ${statusLine}`;
    }
  } else if (meta.status_code != null) {
    statusLine = `HTTP/1.1 ${meta.status_code}`;
  }
  return formatCurlHeaders(
    '<',
    normalizeHeaderMap(parsed.upstream_response_headers),
    statusLine,
  );
}

function buildLogDetailMetaBlock(log, parsed, t) {
  const lines = [];
  const clientIp = parsed?.client_ip || log.ip;
  if (clientIp) {
    lines.push(
      `${detailLabel(t, 'log.detail.client_ip', '客户端 IP')}: ${clientIp}`,
    );
  }
  if (parsed?.xff) {
    lines.push(`X-Forwarded-For: ${parsed.xff}`);
  }
  if (parsed?.request_body_bytes != null) {
    lines.push(
      `${detailLabel(t, 'log.detail.request_body_bytes', '请求体大小')}: ${parsed.request_body_bytes} bytes`,
    );
  }
  if (parsed?.response_body_bytes != null) {
    lines.push(
      `${detailLabel(t, 'log.detail.response_body_bytes', '响应体大小')}: ${parsed.response_body_bytes} bytes`,
    );
  }
  return lines.join('\n');
}

function getLogDetailParts(log) {
  const parsed = parseLogOther(log.other);
  const userInput =
    parsed?.user_input != null ? String(parsed.user_input).trim() : '';
  const content = log.content != null ? String(log.content).trim() : '';
  const badges = [];
  if (log.use_time > 0) badges.push(`${log.use_time}s`);
  if (log.elapsed_time != null && log.elapsed_time !== '')
    badges.push(`${log.elapsed_time}ms`);
  if (log.is_stream) badges.push('Stream');
  if (log.system_prompt_reset) badges.push('System Prompt Reset');

  return { parsed, userInput, content, badges };
}

/** 弹窗完整详情（HTTP 头 curl 格式 + 用户输入 + 计费行） */
function buildLogDetailFullText(log, t) {
  const { parsed, userInput, content, badges } = getLogDetailParts(log);
  const parts = [];
  const label = userInputLabel(t);

  appendDetailSection(
    parts,
    detailLabel(t, 'log.detail.meta', '连接信息'),
    buildLogDetailMetaBlock(log, parsed, t),
  );
  appendDetailSection(
    parts,
    detailLabel(t, 'log.detail.client_request_headers', '客户端请求 Header'),
    buildClientRequestCurl(parsed),
  );
  appendDetailSection(
    parts,
    detailLabel(t, 'log.detail.client_response_headers', '响应客户端 Header'),
    buildClientResponseCurl(parsed),
  );
  appendDetailSection(
    parts,
    detailLabel(t, 'log.detail.upstream_request_headers', '上游请求 Header'),
    buildUpstreamRequestCurl(parsed),
  );
  appendDetailSection(
    parts,
    detailLabel(t, 'log.detail.upstream_response_headers', '上游响应 Header'),
    buildUpstreamResponseCurl(parsed),
  );

  if (userInput) {
    parts.push(`${label}\n${userInput}`);
  }
  if (content) {
    parts.push(content);
  }
  if (badges.length) {
    parts.push(badges.join(' · '));
  }

  if (parsed) {
    const rest = {};
    Object.entries(parsed).forEach(([k, v]) => {
      if (!LOG_OTHER_KNOWN_KEYS.has(k)) {
        rest[k] = v;
      }
    });
    if (Object.keys(rest).length > 0) {
      const hdrLabel = detailLabel(t, 'log.table.upstream_meta', '其他元数据');
      parts.push(`${hdrLabel}\n${JSON.stringify(rest, null, 2)}`);
    }
  }
  if (!parsed && log.other != null && String(log.other).trim()) {
    parts.push(String(log.other));
  }
  return parts.join('\n\n');
}

function renderChannelCell(log) {
  if (!log.channel && !log.channel_name) {
    return null;
  }
  return (
    <div className='log-channel-cell'>
      {log.channel ? (
        <Link className='log-channel-id' to={`/channel/edit/${log.channel}`}>
          #{log.channel}
        </Link>
      ) : null}
      {log.channel_name ? (
        <span className='log-channel-name' title={log.channel_name}>
          {log.channel_name}
        </span>
      ) : null}
    </div>
  );
}

function buildLogDetailRatioLine(log) {
  return log.content != null ? String(log.content).trim() : '';
}

function extractHttpStatusCode(parsed) {
  if (!parsed || typeof parsed !== 'object') {
    return null;
  }
  for (const key of [
    'http_status',
    'status_code',
    'upstream_http_status',
    'test_http_status',
  ]) {
    const v = parsed[key];
    if (v == null || v === '') {
      continue;
    }
    const n = Number(v);
    if (!Number.isNaN(n) && n > 0) {
      return n;
    }
    return String(v);
  }
  return null;
}

/** 详情列第二行：HTTP 状态码 + 用时 / 耗时 / Stream 等 */
function buildLogDetailMetaLine(log, t) {
  const { parsed, badges } = getLogDetailParts(log);
  const label =
    t('log.table.http_status') !== 'log.table.http_status'
      ? t('log.table.http_status')
      : 'HTTP状态码';
  const code = extractHttpStatusCode(parsed);
  const tail = badges.join(' · ');

  if (code != null && tail) {
    return `${label} ${code} · ${tail}`;
  }
  if (code != null) {
    return `${label} ${code}`;
  }
  if (tail) {
    return `${label}  ${tail}`;
  }
  return '';
}

function renderLogDetailCompact(log, t, onOpenFull) {
  const full = buildLogDetailFullText(log, t);
  const ratioLine = buildLogDetailRatioLine(log);
  const metaLine = buildLogDetailMetaLine(log, t);

  if (!ratioLine && !metaLine && !full.trim()) {
    return <span className='log-detail-empty'>—</span>;
  }

  const previewTitle = [ratioLine, metaLine].filter(Boolean).join('\n');

  return (
    <div className='log-detail-preview'>
      <div className='log-detail-body' title={previewTitle || full}>
        {ratioLine ? (
          <span className='log-detail-line log-detail-line-ratio'>{ratioLine}</span>
        ) : null}
        {metaLine ? (
          <span className='log-detail-line log-detail-line-meta'>{metaLine}</span>
        ) : null}
      </div>
      <Button
        type='button'
        className='log-detail-expand-btn'
        basic
        compact
        icon='expand'
        aria-label={t('log.table.detail_expand')}
        title={t('log.table.detail_expand')}
        onClick={(e) => {
          e.stopPropagation();
          onOpenFull(full);
        }}
      />
    </div>
  );
}

function renderModelCell(modelName) {
  if (!modelName) {
    return '';
  }
  return (
    <span className='log-model-ellipsis' title={modelName}>
      {renderColorLabel(modelName)}
    </span>
  );
}

function renderNumericCell(v) {
  if (v === null || v === undefined) {
    return <span style={{ color: '#999' }}>—</span>;
  }
  return v;
}

/** 与 index.css 列宽一致；最后一列不设宽，在 table-layout:fixed 下占满剩余空间 */
function LogsTableColGroup({ isAdminUser }) {
  const widths = [
    '12rem', // time (YYYY-MM-DD HH:mm:ss)
    ...(isAdminUser ? ['9.5rem'] : []), // channel
    '4.5rem', // group
    '4.5rem', // type
    '13.5rem', // model
    ...(isAdminUser ? ['5.5rem'] : []), // user
    '6rem', // token
    '4.5rem', // id
    '7.5rem', // ip
    '4.75rem', // prompt tokens
    '4.75rem', // completion tokens
    '5.5rem', // quota
    '3.75rem', // use time
  ];
  return (
    <colgroup>
      {widths.map((w, i) => (
        <col key={i} style={{ width: w }} />
      ))}
      <col className='log-col-detail-fill' />
    </colgroup>
  );
}

const LogsTable = () => {
  const { t } = useTranslation();
  const [logs, setLogs] = useState([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [activePage, setActivePage] = useState(1);
  const [logType, setLogType] = useState(0);
  const [includeErrors, setIncludeErrors] = useState(false);
  const isAdminUser = isAdmin();
  let now = new Date();
  const [inputs, setInputs] = useState({
    username: '',
    token_name: '',
    model_name: '',
    start_timestamp: timestamp2string(now.getTime() / 1000 - 300),
    end_timestamp: timestamp2string(now.getTime() / 1000),
    channel: '',
    group: '',
    request_id: '',
  });
  const {
    username,
    token_name,
    model_name,
    start_timestamp,
    end_timestamp,
    channel,
    group,
    request_id,
  } = inputs;

  const [stat, setStat] = useState({
    quota: 0,
    rpm: 0,
    tpm: 0,
  });
  const [statLoading, setStatLoading] = useState(false);

  const [logDetailModal, setLogDetailModal] = useState({ open: false, body: '' });

  const includeErrorsQuery = () => (includeErrors ? '&include_errors=1' : '');

  const LOG_OPTIONS = [
    { key: '0', text: t('log.type.all'), value: 0 },
    { key: '1', text: t('log.type.topup'), value: 1 },
    { key: '2', text: t('log.type.usage'), value: 2 },
    { key: '3', text: t('log.type.admin'), value: 3 },
    { key: '4', text: t('log.type.system'), value: 4 },
    { key: '5', text: t('log.type.test'), value: 5 },
    { key: '6', text: t('log.type.error'), value: 6 },
    { key: '7', text: t('log.type.refund'), value: 7 },
  ];

  const handleInputChange = (e, { name, value }) => {
    setInputs((inputs) => ({ ...inputs, [name]: value }));
  };

  const getLogSelfStat = async () => {
    let localStartTimestamp = Date.parse(start_timestamp) / 1000;
    let localEndTimestamp = Date.parse(end_timestamp) / 1000;
    const g = encodeURIComponent(group || '');
    const rid = encodeURIComponent(request_id || '');
    let res = await API.get(
      `/api/log/self/stat?type=${logType}&token_name=${encodeURIComponent(token_name)}&model_name=${encodeURIComponent(model_name)}&start_timestamp=${localStartTimestamp}&end_timestamp=${localEndTimestamp}&group=${g}&request_id=${rid}${includeErrorsQuery()}`
    );
    const { success, message, data } = res.data;
    if (success) {
      setStat(data);
    } else {
      showError(message);
    }
  };

  const getLogStat = async () => {
    let localStartTimestamp = Date.parse(start_timestamp) / 1000;
    let localEndTimestamp = Date.parse(end_timestamp) / 1000;
    const g = encodeURIComponent(group || '');
    const rid = encodeURIComponent(request_id || '');
    let res = await API.get(
      `/api/log/stat?type=${logType}&username=${encodeURIComponent(username)}&token_name=${encodeURIComponent(token_name)}&model_name=${encodeURIComponent(model_name)}&start_timestamp=${localStartTimestamp}&end_timestamp=${localEndTimestamp}&channel=${channel}&group=${g}&request_id=${rid}${includeErrorsQuery()}`
    );
    const { success, message, data } = res.data;
    if (success) {
      setStat(data);
    } else {
      showError(message);
    }
  };

  const fetchUsageStat = async () => {
    setStatLoading(true);
    try {
      if (isAdminUser) {
        await getLogStat();
      } else {
        await getLogSelfStat();
      }
    } finally {
      setStatLoading(false);
    }
  };

  const loadLogs = async (page) => {
    let url = '';
    let localStartTimestamp = Date.parse(start_timestamp) / 1000;
    let localEndTimestamp = Date.parse(end_timestamp) / 1000;
    const g = encodeURIComponent(group || '');
    const rid = encodeURIComponent(request_id || '');
    if (isAdminUser) {
      url = `/api/log/?p=${page}&page_size=${ITEMS_PER_PAGE}&type=${logType}&username=${encodeURIComponent(username)}&token_name=${encodeURIComponent(token_name)}&model_name=${encodeURIComponent(model_name)}&start_timestamp=${localStartTimestamp}&end_timestamp=${localEndTimestamp}&channel=${channel}&group=${g}&request_id=${rid}${includeErrorsQuery()}`;
    } else {
      url = `/api/log/self/?p=${page}&page_size=${ITEMS_PER_PAGE}&type=${logType}&token_name=${encodeURIComponent(token_name)}&model_name=${encodeURIComponent(model_name)}&start_timestamp=${localStartTimestamp}&end_timestamp=${localEndTimestamp}&group=${g}&request_id=${rid}${includeErrorsQuery()}`;
    }
    const res = await API.get(url);
    const { success, message, data } = res.data;
    if (success) {
      const items = data && data.items !== undefined ? data.items : data;
      const pageTotal =
        data && typeof data.total === 'number'
          ? data.total
          : Array.isArray(items)
            ? items.length
            : 0;
      setLogs(Array.isArray(items) ? items : []);
      setTotal(pageTotal);
    } else {
      showError(message);
    }
    setLoading(false);
  };

  const onPaginationChange = (e, { activePage }) => {
    (async () => {
      setLoading(true);
      await loadLogs(activePage);
      setActivePage(activePage);
    })();
  };

  const refresh = async () => {
    setLoading(true);
    setActivePage(1);
    await loadLogs(1);
    await fetchUsageStat();
  };

  useEffect(() => {
    refresh().then();
  }, [logType, includeErrors]);

  const sortLog = (key) => {
    if (logs.length === 0) return;
    setLoading(true);
    let sortedLogs = [...logs];
    if (typeof sortedLogs[0][key] === 'string') {
      sortedLogs.sort((a, b) => {
        return ('' + a[key]).localeCompare(b[key]);
      });
    } else {
      sortedLogs.sort((a, b) => {
        if (a[key] === b[key]) return 0;
        if (a[key] > b[key]) return -1;
        if (a[key] < b[key]) return 1;
      });
    }
    if (sortedLogs[0].id === logs[0].id) {
      sortedLogs.reverse();
    }
    setLogs(sortedLogs);
    setLoading(false);
  };

  return (
    <div className='logs-page-body'>
      <Segment secondary style={{ marginBottom: '1rem' }}>
        <div
          style={{
            display: 'flex',
            flexWrap: 'wrap',
            alignItems: 'center',
            gap: '0.75rem 1.25rem',
          }}
        >
          <Header
            as='h3'
            style={{ margin: 0, fontWeight: 600, fontSize: '1.05rem' }}
          >
            {t('log.usage_details')}
          </Header>
          <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
            <span style={{ color: 'rgba(0,0,0,0.55)' }}>
              {t('log.total_quota')}：
            </span>
            {statLoading ? (
              <Loader active inline size='small' />
            ) : (
              <strong>{renderQuota(stat.quota, t)}</strong>
            )}
            <span style={{ color: 'rgba(0,0,0,0.45)', fontSize: '0.92rem' }}>
              RPM {stat.rpm ?? 0} · TPM {stat.tpm ?? 0}
            </span>
          </div>
        </div>
      </Segment>
      <Form>
        <Form.Group>
          <Form.Input
            fluid
            label={t('log.table.token_name')}
            size={'small'}
            width={3}
            value={token_name}
            placeholder={t('log.table.token_name_placeholder')}
            name='token_name'
            onChange={handleInputChange}
          />
          <Form.Input
            fluid
            label={t('log.table.model_name')}
            size={'small'}
            width={3}
            value={model_name}
            placeholder={t('log.table.model_name_placeholder')}
            name='model_name'
            onChange={handleInputChange}
          />
          <Form.Input
            fluid
            label={t('log.table.start_time')}
            size={'small'}
            width={4}
            value={start_timestamp}
            type='datetime-local'
            name='start_timestamp'
            onChange={handleInputChange}
          />
          <Form.Input
            fluid
            label={t('log.table.end_time')}
            size={'small'}
            width={4}
            value={end_timestamp}
            type='datetime-local'
            name='end_timestamp'
            onChange={handleInputChange}
          />
          <Form.Button
            fluid
            label={t('log.buttons.query')}
            size={'small'}
            width={2}
            onClick={refresh}
          >
            {t('log.buttons.submit')}
          </Form.Button>
        </Form.Group>
        {isAdminUser && (
          <>
            <Form.Group>
              <Form.Input
                fluid
                label={t('log.table.channel_id')}
                size={'small'}
                width={3}
                value={channel}
                placeholder={t('log.table.channel_id_placeholder')}
                name='channel'
                onChange={handleInputChange}
              />
              <Form.Input
                fluid
                label={t('log.table.username')}
                size={'small'}
                width={3}
                value={username}
                placeholder={t('log.table.username_placeholder')}
                name='username'
                onChange={handleInputChange}
                {...noAutofillTextProps}
              />
            </Form.Group>
          </>
        )}
        <Form.Group>
          <Form.Input
            fluid
            label="分组"
            size={'small'}
            width={3}
            value={group}
            placeholder="user group"
            name="group"
            onChange={handleInputChange}
          />
          <Form.Input
            fluid
            label="请求 ID"
            size={'small'}
            width={4}
            value={request_id}
            placeholder="request_id 精确匹配"
            name="request_id"
            onChange={handleInputChange}
          />
        </Form.Group>
      </Form>
      <div className='logs-table-wrap'>
      <div className='logs-table-scroll'>
      <Table
        unstackable
        compact
        size='small'
        className='logs-data-table'
      >
        <LogsTableColGroup isAdminUser={isAdminUser} />
        <Table.Header>
          <Table.Row>
            <Table.HeaderCell
              className='log-col-time'
              style={{ cursor: 'pointer' }}
              onClick={() => {
                sortLog('created_at');
              }}
            >
              {t('log.table.time')}
            </Table.HeaderCell>
            {isAdminUser && (
              <Table.HeaderCell
                className='log-col-channel'
                style={{ cursor: 'pointer' }}
                onClick={() => {
                  sortLog('channel');
                }}
              >
                {t('log.table.channel')}
              </Table.HeaderCell>
            )}
            <Table.HeaderCell className='log-col-compact'>分组</Table.HeaderCell>
            <Table.HeaderCell
              className='log-col-compact'
              style={{ cursor: 'pointer' }}
              onClick={() => {
                sortLog('type');
              }}
            >
              {t('log.table.type')}
            </Table.HeaderCell>
            <Table.HeaderCell
              className='log-col-model'
              style={{ cursor: 'pointer' }}
              onClick={() => {
                sortLog('model_name');
              }}
            >
              {t('log.table.model')}
            </Table.HeaderCell>
            {isAdminUser && (
              <Table.HeaderCell
                className='log-col-user'
                style={{ cursor: 'pointer' }}
                onClick={() => {
                  sortLog('username');
                }}
              >
                {t('log.table.username')}
              </Table.HeaderCell>
            )}
            <Table.HeaderCell
              className='log-col-token'
              style={{ cursor: 'pointer' }}
              onClick={() => {
                sortLog('token_name');
              }}
            >
              {t('log.table.token_name')}
            </Table.HeaderCell>
            <Table.HeaderCell
              className='log-col-compact'
              style={{ cursor: 'pointer' }}
              onClick={() => {
                sortLog('token_id');
              }}
            >
              ID
            </Table.HeaderCell>
            <Table.HeaderCell
              className='log-col-ip'
              style={{ cursor: 'pointer' }}
              onClick={() => {
                sortLog('ip');
              }}
            >
              IP
            </Table.HeaderCell>
            <Table.HeaderCell
              className='log-col-num'
              style={{ cursor: 'pointer' }}
              onClick={() => {
                sortLog('prompt_tokens');
              }}
              title={t('log.table.prompt_tokens')}
            >
              {t('log.table.prompt_tokens_short')}
            </Table.HeaderCell>
            <Table.HeaderCell
              className='log-col-num'
              style={{ cursor: 'pointer' }}
              onClick={() => {
                sortLog('completion_tokens');
              }}
              title={t('log.table.completion_tokens')}
            >
              {t('log.table.completion_tokens_short')}
            </Table.HeaderCell>
            <Table.HeaderCell
              className='log-col-quota'
              style={{ cursor: 'pointer' }}
              onClick={() => {
                sortLog('quota');
              }}
            >
              {t('log.table.quota')}
            </Table.HeaderCell>
                <Table.HeaderCell
                  className='log-col-use-time'
                  style={{ cursor: 'pointer' }}
                  onClick={() => {
                    sortLog('use_time');
                  }}
                >
                  {t('log.table.use_time')}
                </Table.HeaderCell>
            <Table.HeaderCell className='log-col-detail'>
              {t('log.table.detail')}
            </Table.HeaderCell>
          </Table.Row>
        </Table.Header>

        <Table.Body>
          {logs.map((log, idx) => {
              if (log.deleted) return <></>;
              return (
                <Table.Row key={`${log.id}-${idx}`}>
                  <Table.Cell className='log-col-time'>
                    {renderTimestamp(log.created_at, log.request_id)}
                  </Table.Cell>
                  {isAdminUser && (
                    <Table.Cell className='log-col-channel'>
                      {renderChannelCell(log)}
                    </Table.Cell>
                  )}
                  <Table.Cell className='log-col-compact'>
                    {log.group ? (
                      <Label basic size='mini'>
                        {log.group}
                      </Label>
                    ) : (
                      ''
                    )}
                  </Table.Cell>
                  <Table.Cell className='log-col-compact'>
                    {renderType(log.type)}
                  </Table.Cell>
                  <Table.Cell className='log-col-model'>
                    {renderModelCell(log.model_name)}
                  </Table.Cell>
                  {isAdminUser && (
                    <Table.Cell className='log-col-user'>
                      {log.username ? (
                        <Link
                          className='log-user-link'
                          to={`/user/edit/${log.user_id}`}
                        >
                          {log.username}
                        </Link>
                      ) : (
                        ''
                      )}
                    </Table.Cell>
                  )}
                  <Table.Cell className='log-col-token'>
                    {log.token_name ? renderColorLabel(log.token_name) : ''}
                  </Table.Cell>
                  <Table.Cell className='log-col-compact'>
                    {log.token_id ? log.token_id : ''}
                  </Table.Cell>
                  <Table.Cell className='log-col-ip'>
                    {log.ip ? (
                      <code className='log-ip'>{log.ip}</code>
                    ) : (
                      ''
                    )}
                  </Table.Cell>
                  <Table.Cell className='log-col-num' textAlign='right'>
                    {renderNumericCell(log.prompt_tokens)}
                  </Table.Cell>
                  <Table.Cell className='log-col-num' textAlign='right'>
                    {renderNumericCell(log.completion_tokens)}
                  </Table.Cell>
                  <Table.Cell className='log-col-quota' textAlign='right'>
                    {log.quota != null
                      ? renderQuota(log.quota, t, 4)
                      : renderNumericCell(undefined)}
                  </Table.Cell>
                      <Table.Cell
                        className='log-col-use-time'
                        textAlign='right'
                      >
                        {renderNumericCell(log.use_time)}
                      </Table.Cell>

                  <Table.Cell className='log-col-detail'>
                    {renderLogDetailCompact(log, t, (body) =>
                      setLogDetailModal({ open: true, body }),
                    )}
                  </Table.Cell>
                </Table.Row>
              );
            })}
        </Table.Body>
      </Table>
      </div>
      <div className='logs-table-toolbar'>
        <div className='logs-table-toolbar-left'>
          <Select
            className='logs-type-select'
            placeholder={t('log.type.select')}
            options={LOG_OPTIONS}
            name='logType'
            value={logType}
            upward
            onChange={(e, { value }) => {
              setLogType(value);
            }}
          />
          <Form.Checkbox
            className='logs-include-errors'
            checked={includeErrors}
            label={t('log.show_failed')}
            onChange={(_, { checked }) => setIncludeErrors(!!checked)}
          />
          <Button size='small' onClick={refresh} loading={loading}>
            {t('log.buttons.refresh')}
          </Button>
        </div>
        <Pagination
          className='logs-table-pagination'
          activePage={activePage}
          onPageChange={onPaginationChange}
          size='small'
          siblingRange={1}
          totalPages={Math.max(1, Math.ceil(total / ITEMS_PER_PAGE))}
        />
      </div>
      </div>

      <Modal
        open={logDetailModal.open}
        onClose={() => setLogDetailModal({ open: false, body: '' })}
        size='large'
        closeOnDimmerClick
      >
        <Modal.Header>{t('log.table.detail')}</Modal.Header>
        <Modal.Content scrolling>
          <pre className='log-detail-modal-body'>{logDetailModal.body}</pre>
        </Modal.Content>
        <Modal.Actions>
          <Button
            onClick={async () => {
              if (await copy(logDetailModal.body)) {
                showSuccess(t('log.table.detail_copied'));
              } else {
                showWarning(t('log.table.detail_copy_failed'));
              }
            }}
          >
            {t('log.table.detail_copy')}
          </Button>
          <Button
            primary
            onClick={() => setLogDetailModal({ open: false, body: '' })}
          >
            {t('log.table.detail_close')}
          </Button>
        </Modal.Actions>
      </Modal>
    </div>
  );
};

export default LogsTable;
