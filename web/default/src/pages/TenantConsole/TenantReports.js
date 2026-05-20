import React, { useEffect, useState } from 'react';
import { Button, Card, Form, Table } from 'semantic-ui-react';
import { API, showError } from '../../helpers';

const TenantReports = () => {
  const [reports, setReports] = useState([]);
  const [loading, setLoading] = useState(false);
  
  const [startTime, setStartTime] = useState(
    new Date(new Date().setDate(new Date().getDate() - 30)).toISOString().slice(0, 10)
  );
  const [endTime, setEndTime] = useState(
    new Date().toISOString().slice(0, 10)
  );

  const loadReports = async () => {
    setLoading(true);
    const startTs = Math.floor(new Date(startTime).getTime() / 1000);
    const endTs = Math.floor(new Date(endTime).getTime() / 1000) + 86400;

    const res = await API.get(`/api/tenant_console/reports/billing?start_time=${startTs}&end_time=${endTs}`);
    if (res.data.success) {
      setReports(res.data.data || []);
    } else {
      // 避免代管模式下未选择租户时弹红框报错，如果报错包含“请先在页面顶部选择”，则静默处理
      if (String(res.data.message).includes('请先在页面顶部选择')) {
        setReports([]);
      } else {
        showError(res.data.message);
      }
    }
    setLoading(false);
  };

  const exportCSV = () => {
    const startTs = Math.floor(new Date(startTime).getTime() / 1000);
    const endTs = Math.floor(new Date(endTime).getTime() / 1000) + 86400;
    window.open(`/api/tenant_console/reports/billing/export?start_time=${startTs}&end_time=${endTs}`, '_blank');
  };

  useEffect(() => {
    loadReports();
  }, []);

  return (
    <div className='dashboard-container'>
      <Card fluid className='chart-card'>
        <Card.Content>
          <Card.Header className='header'>账单与报表</Card.Header>
          <Form>
            <Form.Group widths='equal'>
              <Form.Input
                label='开始日期'
                type='date'
                value={startTime}
                onChange={(e, { value }) => setStartTime(value)}
              />
              <Form.Input
                label='结束日期'
                type='date'
                value={endTime}
                onChange={(e, { value }) => setEndTime(value)}
              />
              <Form.Button label='&nbsp;' primary onClick={loadReports} loading={loading}>查询</Form.Button>
              <Form.Button label='&nbsp;' onClick={exportCSV}>导出 CSV 账单</Form.Button>
            </Form.Group>
          </Form>
          <Table celled>
            <Table.Header>
              <Table.Row>
                <Table.HeaderCell>用户 ID</Table.HeaderCell>
                <Table.HeaderCell>用户名</Table.HeaderCell>
                <Table.HeaderCell>渠道 ID</Table.HeaderCell>
                <Table.HeaderCell>渠道名</Table.HeaderCell>
                <Table.HeaderCell>模型名称</Table.HeaderCell>
                <Table.HeaderCell>消费额度 (Quota)</Table.HeaderCell>
                <Table.HeaderCell>请求次数</Table.HeaderCell>
                <Table.HeaderCell>提示 Token</Table.HeaderCell>
                <Table.HeaderCell>补全 Token</Table.HeaderCell>
              </Table.Row>
            </Table.Header>
            <Table.Body>
              {reports.map((r, i) => (
                <Table.Row key={i}>
                  <Table.Cell>{r.user_id}</Table.Cell>
                  <Table.Cell>{r.username}</Table.Cell>
                  <Table.Cell>{r.channel_id}</Table.Cell>
                  <Table.Cell>{r.channel_name}</Table.Cell>
                  <Table.Cell>{r.model_name}</Table.Cell>
                  <Table.Cell>{r.quota}</Table.Cell>
                  <Table.Cell>{r.request_count}</Table.Cell>
                  <Table.Cell>{r.prompt_tokens}</Table.Cell>
                  <Table.Cell>{r.completion_tokens}</Table.Cell>
                </Table.Row>
              ))}
            </Table.Body>
          </Table>
        </Card.Content>
      </Card>
    </div>
  );
};

export default TenantReports;