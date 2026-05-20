import React, { useEffect, useState } from 'react';
import { Button, Card, Table, Tag } from 'semantic-ui-react';
import { API, showError, showSuccess, timestamp2string } from '../../helpers';

const TenantUpgrades = () => {
  const [requests, setRequests] = useState([]);
  const [loading, setLoading] = useState(false);

  const loadRequests = async () => {
    setLoading(true);
    const res = await API.get('/api/platform/tenants/upgrades');
    const { success, message, data } = res.data;
    if (success) {
      setRequests(data || []);
    } else {
      showError(message);
    }
    setLoading(false);
  };

  useEffect(() => {
    loadRequests();
  }, []);

  const approveRequest = async (id) => {
    const res = await API.post(`/api/platform/tenants/upgrades/${id}/approve`);
    if (res.data.success) {
      showSuccess('审批通过，租户已创建');
      loadRequests();
    } else {
      showError(res.data.message);
    }
  };

  const rejectRequest = async (id) => {
    const res = await API.post(`/api/platform/tenants/upgrades/${id}/reject`);
    if (res.data.success) {
      showSuccess('已拒绝');
      loadRequests();
    } else {
      showError(res.data.message);
    }
  };

  const renderStatus = (status) => {
    switch (status) {
      case 0:
        return <span style={{ color: 'orange' }}>待审核</span>;
      case 1:
        return <span style={{ color: 'green' }}>已通过</span>;
      case 2:
        return <span style={{ color: 'red' }}>已拒绝</span>;
      default:
        return '未知';
    }
  };

  return (
    <div className='dashboard-container'>
      <Card fluid className='chart-card'>
        <Card.Content>
          <Card.Header className='header'>租户升级申请</Card.Header>
          <Table celled>
            <Table.Header>
              <Table.Row>
                <Table.HeaderCell>ID</Table.HeaderCell>
                <Table.HeaderCell>用户 ID</Table.HeaderCell>
                <Table.HeaderCell>企业名称</Table.HeaderCell>
                <Table.HeaderCell>租户标识 (Slug)</Table.HeaderCell>
                <Table.HeaderCell>备注</Table.HeaderCell>
                <Table.HeaderCell>状态</Table.HeaderCell>
                <Table.HeaderCell>申请时间</Table.HeaderCell>
                <Table.HeaderCell>操作</Table.HeaderCell>
              </Table.Row>
            </Table.Header>

            <Table.Body>
              {requests.map((req) => (
                <Table.Row key={req.id}>
                  <Table.Cell>{req.id}</Table.Cell>
                  <Table.Cell>{req.user_id}</Table.Cell>
                  <Table.Cell>{req.name}</Table.Cell>
                  <Table.Cell>{req.slug}</Table.Cell>
                  <Table.Cell>{req.remark}</Table.Cell>
                  <Table.Cell>{renderStatus(req.status)}</Table.Cell>
                  <Table.Cell>{timestamp2string(req.created_time)}</Table.Cell>
                  <Table.Cell>
                    {req.status === 0 && (
                      <>
                        <Button
                          size='mini'
                          positive
                          onClick={() => approveRequest(req.id)}
                        >
                          通过
                        </Button>
                        <Button
                          size='mini'
                          negative
                          onClick={() => rejectRequest(req.id)}
                        >
                          拒绝
                        </Button>
                      </>
                    )}
                  </Table.Cell>
                </Table.Row>
              ))}
            </Table.Body>
          </Table>
        </Card.Content>
      </Card>
    </div>
  );
};

export default TenantUpgrades;