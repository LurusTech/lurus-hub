import React from 'react';
import { Spin } from '@douyinfe/semi-ui';
import SettingsHeaderNavModules from '../../pages/Setting/Operation/SettingsHeaderNavModules';
import SettingsSidebarModulesAdmin from '../../pages/Setting/Operation/SettingsSidebarModulesAdmin';
import useSettingsOptions from '../../pages/Setting/useSettingsOptions';

const INITIAL_STATE = {
  HeaderNavModules: '',
  SidebarModulesAdmin: '',
};

const UIModulesSettingPage = () => {
  const { inputs, loading, refresh } = useSettingsOptions(INITIAL_STATE);

  return (
    <Spin spinning={loading} size='large'>
      <div style={{ marginTop: '10px' }}>
        <SettingsHeaderNavModules options={inputs} refresh={refresh} />
      </div>
      <div style={{ marginTop: '10px' }}>
        <SettingsSidebarModulesAdmin options={inputs} refresh={refresh} />
      </div>
    </Spin>
  );
};

export default UIModulesSettingPage;
