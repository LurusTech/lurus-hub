import React from 'react';
import ChatsSetting from './ChatsSetting';
import DrawingSetting from './DrawingSetting';

const AIFeaturesSettingPage = () => {
  return (
    <>
      <ChatsSetting />
      <div style={{ marginTop: '10px' }}>
        <DrawingSetting />
      </div>
    </>
  );
};

export default AIFeaturesSettingPage;
