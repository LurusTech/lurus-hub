import React from 'react';
import ModelSetting from './ModelSetting';
import ModelDeploymentSetting from './ModelDeploymentSetting';

const ModelConfigSettingPage = () => {
  return (
    <>
      <ModelSetting />
      <div style={{ marginTop: '10px' }}>
        <ModelDeploymentSetting />
      </div>
    </>
  );
};

export default ModelConfigSettingPage;
