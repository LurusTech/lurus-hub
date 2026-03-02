import React from 'react';
import { Card, Spin } from '@douyinfe/semi-ui';
import SettingsCreditLimit from '../../pages/Setting/Operation/SettingsCreditLimit';
import SettingsCheckin from '../../pages/Setting/Operation/SettingsCheckin';
import RateLimitSetting from './RateLimitSetting';
import useSettingsOptions from '../../pages/Setting/useSettingsOptions';

const INITIAL_STATE = {
  QuotaForNewUser: 0,
  PreConsumedQuota: 0,
  QuotaForInviter: 0,
  QuotaForInvitee: 0,
  'quota_setting.enable_free_model_pre_consume': true,
  'checkin_setting.enabled': false,
  'checkin_setting.min_quota': 1000,
  'checkin_setting.max_quota': 10000,
};

const QuotaLimitsSettingPage = () => {
  const { inputs, loading, refresh } = useSettingsOptions(INITIAL_STATE);

  return (
    <Spin spinning={loading} size="large">
      <Card style={{ marginTop: '10px' }}>
        <SettingsCreditLimit options={inputs} refresh={refresh} />
      </Card>
      <Card style={{ marginTop: '10px' }}>
        <SettingsCheckin options={inputs} refresh={refresh} />
      </Card>
      <div style={{ marginTop: '10px' }}>
        <RateLimitSetting />
      </div>
    </Spin>
  );
};

export default QuotaLimitsSettingPage;
