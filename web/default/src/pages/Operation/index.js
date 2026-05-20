import React from 'react';
import { Navigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Card } from 'semantic-ui-react';
import OperationSetting from '../../components/OperationSetting';
import { isRoot } from '../../helpers';

const OperationPage = () => {
  const { t } = useTranslation();

  if (!isRoot()) {
    return <Navigate to='/' replace />;
  }

  return (
    <div className='dashboard-container'>
      <Card fluid className='chart-card'>
        <Card.Content>
          <Card.Header className='header'>
            {t('header.operations_management')}
          </Card.Header>
          <OperationSetting />
        </Card.Content>
      </Card>
    </div>
  );
};

export default OperationPage;
