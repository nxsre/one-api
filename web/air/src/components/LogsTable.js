import React, { useEffect, useState } from 'react';
import { API, copy, isAdmin, showError, showSuccess, showWarning, timestamp2string } from '../helpers';

import { Avatar, Button, Form, Layout, Modal, Select, Space, Spin, Table, Tag } from '@douyinfe/semi-ui';
import { ITEMS_PER_PAGE } from '../constants';
import { renderNumber, renderQuota, stringToColor } from '../helpers/render';
import Paragraph from '@douyinfe/semi-ui/lib/es/typography/paragraph';

const { Header } = Layout;

function renderTimestamp(timestamp) {
  return (<>
    {timestamp2string(timestamp)}
  </>);
}

const MODE_OPTIONS = [{ key: 'all', text: '全部用户', value: 'all' }, { key: 'self', text: '当前用户', value: 'self' }];

const colors = ['amber', 'blue', 'cyan', 'green', 'grey', 'indigo', 'light-blue', 'lime', 'orange', 'pink', 'purple', 'red', 'teal', 'violet', 'yellow'];

function renderType(type) {
  switch (type) {
    case 1:
      return <Tag color="cyan" size="large"> 充值 </Tag>;
    case 2:
      return <Tag color="lime" size="large"> 消费 </Tag>;
    case 3:
      return <Tag color="orange" size="large"> 管理 </Tag>;
    case 4:
      return <Tag color="purple" size="large"> 系统 </Tag>;
    case 5:
      return <Tag color="violet" size="large"> 测试 </Tag>;
    case 6:
      return <Tag color="red" size="large"> 错误 </Tag>;
    case 7:
      return <Tag color="cyan" size="large"> 退款 </Tag>;
    default:
      return <Tag color="black" size="large"> 未知 </Tag>;
  }
}

function renderIsStream(bool) {
  if (bool) {
    return <Tag color="blue" size="large">流</Tag>;
  } else {
    return <Tag color="purple" size="large">非流</Tag>;
  }
}

function renderUseTime(type) {
  const time = parseInt(type);
  if (time < 101) {
    return <Tag color="green" size="large"> {time} s </Tag>;
  } else if (time < 300) {
    return <Tag color="orange" size="large"> {time} s </Tag>;
  } else {
    return <Tag color="red" size="large"> {time} s </Tag>;
  }
}

const LogsTable = () => {
  const columns = [{
    title: '时间', dataIndex: 'timestamp2string'
  }, {
    title: '渠道',
    dataIndex: 'channel',
    className: isAdmin() ? 'tableShow' : 'tableHiddle',
    render: (text, record, index) => {
      return (isAdminUser ? record.type === 0 || record.type === 2 ? <div>
        {<Tag color={colors[parseInt(text) % colors.length]} size="large"> {text} </Tag>}
      </div> : <></> : <></>);
    }
  }, {
    title: '用户',
    dataIndex: 'username',
    className: isAdmin() ? 'tableShow' : 'tableHiddle',
    render: (text, record, index) => {
      return (isAdminUser ? <div>
        <Avatar size="small" color={stringToColor(text)} style={{ marginRight: 4 }}
          onClick={() => showUserInfo(record.user_id)}>
          {typeof text === 'string' && text.slice(0, 1)}
        </Avatar>
        {text}
      </div> : <></>);
    }
  }, {
    title: '令牌', dataIndex: 'token_name', render: (text, record, index) => {
      return (record.type === 0 || record.type === 2 ? <div>
        <Tag color="grey" size="large" onClick={() => {
          copyText(text);
        }}> {text} </Tag>
      </div> : <></>);
    }
  }, {
    title: '类型', dataIndex: 'type', render: (text, record, index) => {
      return (<div>
        {renderType(text)}
      </div>);
    }
  }, {
    title: '模型', dataIndex: 'model_name', render: (text, record, index) => {
      return (record.type === 0 || record.type === 2 ? <div>
        <Tag color={stringToColor(text)} size="large" onClick={() => {
          copyText(text);
        }}> {text} </Tag>
      </div> : <></>);
    }
  },
  // {
  //   title: '用时', dataIndex: 'use_time', render: (text, record, index) => {
  //     return (<div>
  //       <Space>
  //         {renderUseTime(text)}
  //         {renderIsStream(record.is_stream)}
  //       </Space>
  //     </div>);
  //   }
  // },
  {
    title: '提示', dataIndex: 'prompt_tokens', render: (text, record, index) => {
      return (record.type === 0 || record.type === 2 ? <div>
        {<span> {text} </span>}
      </div> : <></>);
    }
  }, {
    title: '补全', dataIndex: 'completion_tokens', render: (text, record, index) => {
      return (parseInt(text) > 0 && (record.type === 0 || record.type === 2) ? <div>
        {<span> {text} </span>}
      </div> : <></>);
    }
  }, {
    title: '花费', dataIndex: 'quota', render: (text, record, index) => {
      return (record.type === 0 || record.type === 2 ? <div>
        {renderQuota(text, 6)}
      </div> : <></>);
    }
  }, {
    title: '详情', dataIndex: 'content', render: (text, record, index) => {
      return <Paragraph ellipsis={{ rows: 2, showTooltip: { type: 'popover', opts: { style: { width: 240 } } } }}
        style={{ maxWidth: 240 }}>
        {text}
      </Paragraph>;
    }
  }];

  const [logs, setLogs] = useState([]);
  const [showStat, setShowStat] = useState(false);
  const [loading, setLoading] = useState(false);
  const [loadingStat, setLoadingStat] = useState(false);
  const [activePage, setActivePage] = useState(1);
  const [total, setTotal] = useState(0);
  const [pageSize, setPageSize] = useState(ITEMS_PER_PAGE);
  const [searchKeyword, setSearchKeyword] = useState('');
  const [searching, setSearching] = useState(false);
  const [logType, setLogType] = useState(0);
  const isAdminUser = isAdmin();
  let now = new Date();
  // 默认展示最近 5 分钟
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
  const { username, token_name, model_name, start_timestamp, end_timestamp, channel, group, request_id } = inputs;

  const [stat, setStat] = useState({
    quota: 0, rpm: 0, tpm: 0,
  });

  const handleInputChange = (value, name) => {
    setInputs((inputs) => ({ ...inputs, [name]: value }));
  };

  const getLogSelfStat = async () => {
    let localStartTimestamp = Date.parse(start_timestamp) / 1000;
    let localEndTimestamp = Date.parse(end_timestamp) / 1000;
    let res = await API.get(`/api/log/self/stat?type=${logType}&token_name=${encodeURIComponent(token_name)}&model_name=${encodeURIComponent(model_name)}&start_timestamp=${localStartTimestamp}&end_timestamp=${localEndTimestamp}&group=${encodeURIComponent(group || '')}&request_id=${encodeURIComponent(request_id || '')}`);
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
    let res = await API.get(`/api/log/stat?type=${logType}&username=${encodeURIComponent(username)}&token_name=${encodeURIComponent(token_name)}&model_name=${encodeURIComponent(model_name)}&start_timestamp=${localStartTimestamp}&end_timestamp=${localEndTimestamp}&channel=${channel}&group=${encodeURIComponent(group || '')}&request_id=${encodeURIComponent(request_id || '')}`);
    const { success, message, data } = res.data;
    if (success) {
      setStat(data);
    } else {
      showError(message);
    }
  };

  const handleEyeClick = async () => {
    setLoadingStat(true);
    if (isAdminUser) {
      await getLogStat();
    } else {
      await getLogSelfStat();
    }
    setShowStat(true);
    setLoadingStat(false);
  };

  const showUserInfo = async (userId) => {
    if (!isAdminUser) {
      return;
    }
    const res = await API.get(`/api/user/${userId}`);
    const { success, message, data } = res.data;
    if (success) {
      Modal.info({
        title: '用户信息', content: <div style={{ padding: 12 }}>
          <p>用户名: {data.username}</p>
          <p>余额: {renderQuota(data.quota)}</p>
          <p>已用额度：{renderQuota(data.used_quota)}</p>
          <p>请求次数：{renderNumber(data.request_count)}</p>
        </div>, centered: true
      });
    } else {
      showError(message);
    }
  };

  const setLogsFormat = (items) => {
    const arr = Array.isArray(items) ? items : [];
    for (let i = 0; i < arr.length; i++) {
      arr[i].timestamp2string = timestamp2string(arr[i].created_at);
      arr[i].key = '' + arr[i].id;
    }
    setLogs(arr);
  };

  const loadLogs = async (page, pageSize, logTypeParam) => {
    setLoading(true);
    const lt = logTypeParam !== undefined ? logTypeParam : logType;
    let localStartTimestamp = Date.parse(start_timestamp) / 1000;
    let localEndTimestamp = Date.parse(end_timestamp) / 1000;
    const g = encodeURIComponent(group || '');
    const rid = encodeURIComponent(request_id || '');
    let url = '';
    if (isAdminUser) {
      url = `/api/log/?p=${page}&page_size=${pageSize}&type=${lt}&username=${encodeURIComponent(username)}&token_name=${encodeURIComponent(token_name)}&model_name=${encodeURIComponent(model_name)}&start_timestamp=${localStartTimestamp}&end_timestamp=${localEndTimestamp}&channel=${channel}&group=${g}&request_id=${rid}`;
    } else {
      url = `/api/log/self/?p=${page}&page_size=${pageSize}&type=${lt}&token_name=${encodeURIComponent(token_name)}&model_name=${encodeURIComponent(model_name)}&start_timestamp=${localStartTimestamp}&end_timestamp=${localEndTimestamp}&group=${g}&request_id=${rid}`;
    }
    const res = await API.get(url);
    const { success, message, data } = res.data;
    if (success) {
      const items = data && data.items !== undefined ? data.items : data;
      const t =
        data && typeof data.total === 'number'
          ? data.total
          : Array.isArray(items)
            ? items.length
            : 0;
      setTotal(t);
      setLogsFormat(items);
    } else {
      showError(message);
    }
    setLoading(false);
  };

  const pageData = logs;

  const handlePageChange = page => {
    setActivePage(page);
    loadLogs(page, pageSize).then();
  };

  const handlePageSizeChange = async (size) => {
    localStorage.setItem('page-size', size + '');
    setPageSize(size);
    setActivePage(1);
    loadLogs(1, size)
      .then()
      .catch((reason) => {
        showError(reason);
      });
  };

  const refresh = async (localLogType) => {
    setActivePage(1);
    await loadLogs(1, pageSize, localLogType);
  };

  useEffect(() => {
    const localPageSize = parseInt(localStorage.getItem('page-size')) || ITEMS_PER_PAGE;
    setPageSize(localPageSize);
    loadLogs(1, localPageSize)
      .then()
      .catch((reason) => {
        showError(reason);
      });
  }, []);

  const searchLogs = async () => {
    showWarning('关键词搜索已废弃，请使用模型名称（支持 % 模糊）、分组、请求 ID 等条件筛选。');
    setSearching(false);
  };

  const copyText = async (text) => {
    if (await copy(text)) {
      showSuccess('已复制：' + text);
    } else {
      Modal.error({ title: '无法复制到剪贴板，请手动复制', content: text });
    }
  };

  return (<>
    <Layout>
      <Header>
        <Spin spinning={loadingStat}>
          <h3>使用明细（总消耗额度：
            <span onClick={handleEyeClick} style={{
              cursor: 'pointer', color: 'gray'
            }}>{showStat ? (<>{renderQuota(stat.quota)} · RPM {stat.rpm ?? 0} · TPM {stat.tpm ?? 0}</>) : '点击查看'}</span>
            ）
          </h3>
        </Spin>
      </Header>
      <Form layout="horizontal" style={{ marginTop: 10 }}>
        <>
          <Form.Input field="token_name" label="令牌名称" style={{ width: 176 }} value={token_name}
            placeholder={'可选值'} name="token_name"
            onChange={value => handleInputChange(value, 'token_name')} />
          <Form.Input field="model_name" label="模型名称" style={{ width: 176 }} value={model_name}
            placeholder="可选值"
            name="model_name"
            onChange={value => handleInputChange(value, 'model_name')} />
          <Form.DatePicker field="start_timestamp" label="起始时间" style={{ width: 272 }}
            initValue={start_timestamp}
            value={start_timestamp} type="dateTime"
            name="start_timestamp"
            onChange={value => handleInputChange(value, 'start_timestamp')} />
          <Form.DatePicker field="end_timestamp" fluid label="结束时间" style={{ width: 272 }}
            initValue={end_timestamp}
            value={end_timestamp} type="dateTime"
            name="end_timestamp"
            onChange={value => handleInputChange(value, 'end_timestamp')} />
          {isAdminUser && <>
            <Form.Input field="channel" label="渠道 ID" style={{ width: 176 }} value={channel}
              placeholder="可选值" name="channel"
              onChange={value => handleInputChange(value, 'channel')} />
            <Form.Input field="username" label="用户名称" style={{ width: 176 }} value={username}
              placeholder={'可选值'} name="username"
              onChange={value => handleInputChange(value, 'username')} />
          </>}
          <Form.Input field="group" label="分组" style={{ width: 176 }} value={group}
            placeholder="可选值" name="group"
            onChange={value => handleInputChange(value, 'group')} />
          <Form.Input field="request_id" label="请求 ID" style={{ width: 220 }} value={request_id}
            placeholder="精确匹配" name="request_id"
            onChange={value => handleInputChange(value, 'request_id')} />
          <Form.Section>
            <Button label="查询" type="primary" htmlType="submit" className="btn-margin-right"
              onClick={refresh} loading={loading}>查询</Button>
          </Form.Section>
        </>
      </Form>
      <Table style={{ marginTop: 5 }} columns={columns} dataSource={pageData} pagination={{
        currentPage: activePage,
        pageSize: pageSize,
        total: total,
        pageSizeOpts: [10, 20, 50, 100],
        showSizeChanger: true,
        onPageSizeChange: (size) => {
          handlePageSizeChange(size).then();
        },
        onPageChange: handlePageChange
      }} />
      <Select defaultValue="0" style={{ width: 120 }} onChange={(value) => {
        setLogType(parseInt(value));
        refresh(parseInt(value)).then();
      }}>
        <Select.Option value="0">全部</Select.Option>
        <Select.Option value="1">充值</Select.Option>
        <Select.Option value="2">消费</Select.Option>
        <Select.Option value="3">管理</Select.Option>
        <Select.Option value="4">系统</Select.Option>
        <Select.Option value="5">测试</Select.Option>
        <Select.Option value="6">错误</Select.Option>
        <Select.Option value="7">退款</Select.Option>
      </Select>
    </Layout>
  </>);
};

export default LogsTable;
