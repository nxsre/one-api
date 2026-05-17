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
} from '../helpers';
import { useTranslation } from 'react-i18next';

import { ITEMS_PER_PAGE } from '../constants';
import { renderColorLabel, renderQuota } from '../helpers/render';
import { Link } from 'react-router-dom';

function renderTimestamp(timestamp, request_id) {
  return (
    <code
      onClick={async () => {
        if (await copy(request_id)) {
          showSuccess(`已复制请求 ID：${request_id}`);
        } else {
          showWarning(`请求 ID 复制失败：${request_id}`);
        }
      }}
      style={{ cursor: 'pointer' }}
    >
      {timestamp2string(timestamp)}
    </code>
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

/** 日志详情完整文本（用于弹窗与两行预览同源，其它字段不再截断） */
function buildLogDetailFullText(log) {
  const parts = [];
  if (log.content != null && String(log.content).trim() !== '') {
    parts.push(String(log.content));
  }
  const badges = [];
  if (log.use_time > 0) {
    badges.push(`${log.use_time}s`);
  }
  if (log.elapsed_time != null && log.elapsed_time !== '') {
    badges.push(`${log.elapsed_time}ms`);
  }
  if (log.is_stream) {
    badges.push('Stream');
  }
  if (log.system_prompt_reset) {
    badges.push('System Prompt Reset');
  }
  if (badges.length) {
    parts.push(badges.join(' · '));
  }
  if (log.other != null && String(log.other).trim() !== '') {
    parts.push(String(log.other));
  }
  return parts.join('\n\n');
}

function renderLogDetailCompact(log, t, onOpenFull) {
  const full = buildLogDetailFullText(log);
  if (!full.trim()) {
    return null;
  }
  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'row',
        alignItems: 'center',
        gap: 8,
      }}
    >
      <div
        style={{
          flex: 1,
          minWidth: 0,
          fontSize: '0.9em',
          lineHeight: 1.45,
          wordBreak: 'break-word',
          display: '-webkit-box',
          WebkitBoxOrient: 'vertical',
          WebkitLineClamp: 2,
          overflow: 'hidden',
          whiteSpace: 'pre-wrap',
        }}
        title={full}
      >
        {full}
      </div>
      <Button
        type='button'
        basic
        compact
        icon='ellipsis horizontal'
        aria-label={t('log.table.detail_expand')}
        title={t('log.table.detail_expand')}
        onClick={(e) => {
          e.stopPropagation();
          onOpenFull(full);
        }}
        style={{
          flexShrink: 0,
          alignSelf: 'center',
          margin: 0,
          padding: '0.35em 0.45em',
        }}
      />
    </div>
  );
}

function renderNumericCell(v) {
  if (v === null || v === undefined) {
    return <span style={{ color: '#999' }}>—</span>;
  }
  return v;
}

const LogsTable = () => {
  const { t } = useTranslation();
  const [logs, setLogs] = useState([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [activePage, setActivePage] = useState(1);
  const [searchKeyword, setSearchKeyword] = useState('');
  const [searching, setSearching] = useState(false);
  const [logType, setLogType] = useState(0);
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

  const LOG_OPTIONS = [
    { key: '0', text: t('log.type.all'), value: 0 },
    { key: '1', text: t('log.type.topup'), value: 1 },
    { key: '2', text: t('log.type.usage'), value: 2 },
    { key: '3', text: t('log.type.admin'), value: 3 },
    { key: '4', text: t('log.type.system'), value: 4 },
    { key: '5', text: t('log.type.test'), value: 5 },
    { key: '6', text: '错误', value: 6 },
    { key: '7', text: '退款', value: 7 },
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
      `/api/log/self/stat?type=${logType}&token_name=${encodeURIComponent(token_name)}&model_name=${encodeURIComponent(model_name)}&start_timestamp=${localStartTimestamp}&end_timestamp=${localEndTimestamp}&group=${g}&request_id=${rid}`
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
      `/api/log/stat?type=${logType}&username=${encodeURIComponent(username)}&token_name=${encodeURIComponent(token_name)}&model_name=${encodeURIComponent(model_name)}&start_timestamp=${localStartTimestamp}&end_timestamp=${localEndTimestamp}&channel=${channel}&group=${g}&request_id=${rid}`
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
      url = `/api/log/?p=${page}&page_size=${ITEMS_PER_PAGE}&type=${logType}&username=${encodeURIComponent(username)}&token_name=${encodeURIComponent(token_name)}&model_name=${encodeURIComponent(model_name)}&start_timestamp=${localStartTimestamp}&end_timestamp=${localEndTimestamp}&channel=${channel}&group=${g}&request_id=${rid}`;
    } else {
      url = `/api/log/self/?p=${page}&page_size=${ITEMS_PER_PAGE}&type=${logType}&token_name=${encodeURIComponent(token_name)}&model_name=${encodeURIComponent(model_name)}&start_timestamp=${localStartTimestamp}&end_timestamp=${localEndTimestamp}&group=${g}&request_id=${rid}`;
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
  }, [logType]);

  const searchLogs = async () => {
    showWarning('关键词搜索已废弃，请使用模型名称（支持 % 模糊）、分组、请求 ID 等条件筛选后点击查询。');
    setSearching(false);
  };

  const handleKeywordChange = async (e, { value }) => {
    setSearchKeyword(value.trim());
  };

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
    <>
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
        <Form.Input
          icon='search'
          placeholder={t('log.search')}
          value={searchKeyword}
          onChange={(e, { value }) => setSearchKeyword(value)}
        />
      </Form>
      <Table basic='very' compact celled size='small' stackable>
        <Table.Header>
          <Table.Row>
            <Table.HeaderCell
              style={{ cursor: 'pointer' }}
              onClick={() => {
                sortLog('created_at');
              }}
              width={3}
            >
              {t('log.table.time')}
            </Table.HeaderCell>
            {isAdminUser && (
              <Table.HeaderCell
                style={{ cursor: 'pointer' }}
                onClick={() => {
                  sortLog('channel');
                }}
                width={1}
              >
                {t('log.table.channel')}
              </Table.HeaderCell>
            )}
            {isAdminUser && (
              <Table.HeaderCell width={2}>渠道名称</Table.HeaderCell>
            )}
            <Table.HeaderCell width={2}>分组</Table.HeaderCell>
            <Table.HeaderCell
              style={{ cursor: 'pointer' }}
              onClick={() => {
                sortLog('type');
              }}
              width={1}
            >
              {t('log.table.type')}
            </Table.HeaderCell>
            <Table.HeaderCell
              style={{ cursor: 'pointer' }}
              onClick={() => {
                sortLog('model_name');
              }}
              width={2}
            >
              {t('log.table.model')}
            </Table.HeaderCell>
                {isAdminUser && (
                  <Table.HeaderCell
                    style={{ cursor: 'pointer' }}
                    onClick={() => {
                      sortLog('username');
                    }}
                    width={2}
                  >
                    {t('log.table.username')}
                  </Table.HeaderCell>
                )}
                <Table.HeaderCell
                  style={{ cursor: 'pointer' }}
                  onClick={() => {
                    sortLog('token_name');
                  }}
                  width={2}
                >
                  {t('log.table.token_name')}
                </Table.HeaderCell>
                <Table.HeaderCell
                  style={{ cursor: 'pointer' }}
                  onClick={() => {
                    sortLog('token_id');
                  }}
                  width={1}
                >
                  令牌 ID
                </Table.HeaderCell>
                <Table.HeaderCell
                  style={{ cursor: 'pointer' }}
                  onClick={() => {
                    sortLog('ip');
                  }}
                  width={2}
                >
                  IP
                </Table.HeaderCell>
                <Table.HeaderCell
                  style={{ cursor: 'pointer' }}
                  onClick={() => {
                    sortLog('prompt_tokens');
                  }}
                  width={1}
                  title={t('log.table.prompt_tokens')}
                >
                  {t('log.table.prompt_tokens_short')}
                </Table.HeaderCell>
                <Table.HeaderCell
                  style={{ cursor: 'pointer' }}
                  onClick={() => {
                    sortLog('completion_tokens');
                  }}
                  width={1}
                  title={t('log.table.completion_tokens')}
                >
                  {t('log.table.completion_tokens_short')}
                </Table.HeaderCell>
                <Table.HeaderCell
                  style={{ cursor: 'pointer' }}
                  onClick={() => {
                    sortLog('quota');
                  }}
                  width={1}
                >
                  {t('log.table.quota')}
                </Table.HeaderCell>
                <Table.HeaderCell
                  style={{ cursor: 'pointer' }}
                  onClick={() => {
                    sortLog('use_time');
                  }}
                  width={1}
                >
                  {t('log.table.use_time')}
                </Table.HeaderCell>
            <Table.HeaderCell style={{ minWidth: '14rem' }}>
              {t('log.table.detail')}
            </Table.HeaderCell>
          </Table.Row>
        </Table.Header>

        <Table.Body>
          {logs.map((log, idx) => {
              if (log.deleted) return <></>;
              return (
                <Table.Row key={`${log.id}-${idx}`}>
                  <Table.Cell>
                    {renderTimestamp(log.created_at, log.request_id)}
                  </Table.Cell>
                  {isAdminUser && (
                    <Table.Cell>
                      {log.channel ? (
                        <Label
                          basic
                          as={Link}
                          to={`/channel/edit/${log.channel}`}
                        >
                          {log.channel}
                        </Label>
                      ) : (
                        ''
                      )}
                    </Table.Cell>
                  )}
                  {isAdminUser && (
                    <Table.Cell>
                      {log.channel_name ? (
                        <span style={{ wordBreak: 'break-all' }}>
                          {log.channel_name}
                        </span>
                      ) : (
                        ''
                      )}
                    </Table.Cell>
                  )}
                  <Table.Cell>
                    {log.group ? (
                      <Label basic size="mini">
                        {log.group}
                      </Label>
                    ) : (
                      ''
                    )}
                  </Table.Cell>
                  <Table.Cell>{renderType(log.type)}</Table.Cell>
                  <Table.Cell>
                    {log.model_name ? renderColorLabel(log.model_name) : ''}
                  </Table.Cell>
                      {isAdminUser && (
                        <Table.Cell>
                          {log.username ? (
                            <Label
                              basic
                              as={Link}
                              to={`/user/edit/${log.user_id}`}
                            >
                              {log.username}
                            </Label>
                          ) : (
                            ''
                          )}
                        </Table.Cell>
                      )}
                      <Table.Cell>
                        {log.token_name ? renderColorLabel(log.token_name) : ''}
                      </Table.Cell>

                      <Table.Cell>
                        {log.token_id ? log.token_id : ''}
                      </Table.Cell>

                      <Table.Cell>
                        {log.ip ? (
                          <code style={{ fontSize: '0.9em' }}>{log.ip}</code>
                        ) : (
                          ''
                        )}
                      </Table.Cell>

                      <Table.Cell textAlign='right'>
                        {renderNumericCell(log.prompt_tokens)}
                      </Table.Cell>
                      <Table.Cell textAlign='right'>
                        {renderNumericCell(log.completion_tokens)}
                      </Table.Cell>
                      <Table.Cell textAlign='right'>
                        {log.quota != null
                          ? renderQuota(log.quota, t, 6)
                          : renderNumericCell(undefined)}
                      </Table.Cell>
                      <Table.Cell textAlign='right'>
                        {renderNumericCell(log.use_time)}
                      </Table.Cell>

                  <Table.Cell
                    style={{
                      maxWidth: '20rem',
                      verticalAlign: 'middle',
                    }}
                  >
                    {renderLogDetailCompact(log, t, (body) =>
                      setLogDetailModal({ open: true, body }),
                    )}
                  </Table.Cell>
                </Table.Row>
              );
            })}
        </Table.Body>

        <Table.Footer>
          <Table.Row>
            <Table.HeaderCell colSpan={20}>
              <Select
                placeholder={t('log.type.select')}
                options={LOG_OPTIONS}
                style={{ marginRight: '8px' }}
                name='logType'
                value={logType}
                onChange={(e, { name, value }) => {
                  setLogType(value);
                }}
              />
              <Button size='small' onClick={refresh} loading={loading}>
                {t('log.buttons.refresh')}
              </Button>
              <Pagination
                floated='right'
                activePage={activePage}
                onPageChange={onPaginationChange}
                size='small'
                siblingRange={1}
                totalPages={Math.max(1, Math.ceil(total / ITEMS_PER_PAGE))}
              />
            </Table.HeaderCell>
          </Table.Row>
        </Table.Footer>
      </Table>

      <Modal
        open={logDetailModal.open}
        onClose={() => setLogDetailModal({ open: false, body: '' })}
        size='large'
        closeOnDimmerClick
      >
        <Modal.Header>{t('log.table.detail')}</Modal.Header>
        <Modal.Content scrolling>
          <pre
            style={{
              whiteSpace: 'pre-wrap',
              wordBreak: 'break-word',
              margin: 0,
              fontFamily: 'inherit',
              fontSize: '0.92rem',
            }}
          >
            {logDetailModal.body}
          </pre>
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
    </>
  );
};

export default LogsTable;
