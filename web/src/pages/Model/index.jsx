/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import React, { useState, useEffect } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Tabs } from '@douyinfe/semi-ui';
import { List, DollarSign, Cpu, Sparkles, Server } from 'lucide-react';

import ModelsPage from '../../components/table/models';
import RatioSetting from '../../components/settings/RatioSetting';
import ModelConfigSettingPage from '../../components/settings/ModelConfigSettingPage';
import AIFeaturesSettingPage from '../../components/settings/AIFeaturesSettingPage';
import DeploymentsTable from '../../components/table/model-deployments';
import DeploymentAccessGuard from '../../components/model-deployments/DeploymentAccessGuard';
import { useModelDeploymentSettings } from '../../hooks/model-deployments/useModelDeploymentSettings';

const TABS = [
  { key: 'list', labelKey: '模型列表', icon: List },
  { key: 'pricing', labelKey: '定价管理', icon: DollarSign },
  { key: 'config', labelKey: '模型配置', icon: Cpu },
  { key: 'features', labelKey: 'AI 功能', icon: Sparkles },
  { key: 'deployment', labelKey: '模型部署', icon: Server },
];

const VALID_TAB_KEYS = new Set(TABS.map((t) => t.key));
const DEFAULT_TAB = 'list';

// Separate component so the deployment hook is only called when this tab is active
const DeploymentTabContent = () => {
  const {
    loading,
    isIoNetEnabled,
    connectionLoading,
    connectionOk,
    connectionError,
    testConnection,
  } = useModelDeploymentSettings();

  return (
    <DeploymentAccessGuard
      loading={loading}
      isEnabled={isIoNetEnabled}
      connectionLoading={connectionLoading}
      connectionOk={connectionOk}
      connectionError={connectionError}
      onRetry={() => testConnection()}
    >
      <DeploymentsTable />
    </DeploymentAccessGuard>
  );
};

const ModelHub = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const location = useLocation();
  const [activeTab, setActiveTab] = useState(DEFAULT_TAB);

  useEffect(() => {
    const params = new URLSearchParams(location.search);
    const tab = params.get('tab');
    if (tab && VALID_TAB_KEYS.has(tab)) {
      setActiveTab(tab);
    } else if (!tab) {
      setActiveTab(DEFAULT_TAB);
    }
  }, [location.search]);

  const onTabChange = (key) => {
    setActiveTab(key);
    if (key === DEFAULT_TAB) {
      navigate('/console/models', { replace: true });
    } else {
      navigate(`/console/models?tab=${key}`, { replace: true });
    }
  };

  const renderContent = () => {
    switch (activeTab) {
      case 'list':
        return <ModelsPage />;
      case 'pricing':
        return <RatioSetting />;
      case 'config':
        return <ModelConfigSettingPage />;
      case 'features':
        return <AIFeaturesSettingPage />;
      case 'deployment':
        return <DeploymentTabContent />;
      default:
        return <ModelsPage />;
    }
  };

  return (
    <div className='px-2'>
      <Tabs
        type='button'
        activeKey={activeTab}
        onChange={onTabChange}
        style={{ marginBottom: 16 }}
      >
        {TABS.map((tab) => {
          const Icon = tab.icon;
          return (
            <Tabs.TabPane
              key={tab.key}
              itemKey={tab.key}
              tab={
                <span className='flex items-center gap-1.5'>
                  <Icon size={15} />
                  {t(tab.labelKey)}
                </span>
              }
            />
          );
        })}
      </Tabs>
      {renderContent()}
    </div>
  );
};

export default ModelHub;
