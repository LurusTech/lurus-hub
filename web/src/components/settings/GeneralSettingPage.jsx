import React from 'react';
import { Card, Spin } from '@douyinfe/semi-ui';
import SettingsGeneral from '../../pages/Setting/Operation/SettingsGeneral';
import useSettingsOptions from '../../pages/Setting/useSettingsOptions';

const INITIAL_STATE = {
  QuotaForNewUser: 0,
  PreConsumedQuota: 0,
  QuotaForInviter: 0,
  QuotaForInvitee: 0,
  'quota_setting.enable_free_model_pre_consume': true,
  TopUpLink: '',
  'general_setting.docs_link': '',
  QuotaPerUnit: 0,
  USDExchangeRate: 0,
  RetryTimes: 0,
  'general_setting.quota_display_type': 'USD',
  DisplayTokenStatEnabled: false,
  DefaultCollapseSidebar: false,
  DemoSiteEnabled: false,
  SelfUseModeEnabled: false,
};

const GeneralSettingPage = () => {
  const { inputs, loading, refresh } = useSettingsOptions(INITIAL_STATE);

  return (
    <Spin spinning={loading} size='large'>
      <Card style={{ marginTop: '10px' }}>
        <SettingsGeneral options={inputs} refresh={refresh} />
      </Card>
    </Spin>
  );
};

export default GeneralSettingPage;
