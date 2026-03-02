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

import React, { useEffect, useState, lazy, Suspense } from 'react';
import { Layout, Spin } from '@douyinfe/semi-ui';
import { useNavigate, useLocation } from 'react-router-dom';

import SettingsSidebar from './SettingsSidebar';
import { isRoot } from '../../helpers';

import GeneralSettingPage from '../../components/settings/GeneralSettingPage';
import BrandingSettingPage from '../../components/settings/BrandingSettingPage';
import ContentSettingPage from '../../components/settings/ContentSettingPage';
import UIModulesSettingPage from '../../components/settings/UIModulesSettingPage';
import AuthSettingPage from '../../components/settings/AuthSettingPage';
import SecuritySettingPage from '../../components/settings/SecuritySettingPage';
import RatioSetting from '../../components/settings/RatioSetting';
import ModelConfigSettingPage from '../../components/settings/ModelConfigSettingPage';
import AIFeaturesSettingPage from '../../components/settings/AIFeaturesSettingPage';
import QuotaLimitsSettingPage from '../../components/settings/QuotaLimitsSettingPage';
import MonitoringSettingPage from '../../components/settings/MonitoringSettingPage';

// Old tab key → new tab key redirect mapping
const TAB_REDIRECT = {
  operation: 'general',
  dashboard: 'monitoring',
  chats: 'ai-features',
  drawing: 'ai-features',
  ratio: 'pricing',
  ratelimit: 'quota-limits',
  models: 'model-config',
  'model-deployment': 'model-config',
  system: 'auth',
  other: 'branding',
};

const DEFAULT_TAB = 'general';

const renderContent = (activeKey) => {
  switch (activeKey) {
    case 'general':
      return <GeneralSettingPage />;
    case 'branding':
      return <BrandingSettingPage />;
    case 'content':
      return <ContentSettingPage />;
    case 'ui-modules':
      return <UIModulesSettingPage />;
    case 'auth':
      return <AuthSettingPage />;
    case 'security':
      return <SecuritySettingPage />;
    case 'pricing':
      return <RatioSetting />;
    case 'model-config':
      return <ModelConfigSettingPage />;
    case 'ai-features':
      return <AIFeaturesSettingPage />;
    case 'quota-limits':
      return <QuotaLimitsSettingPage />;
    case 'monitoring':
      return <MonitoringSettingPage />;
    default:
      return <GeneralSettingPage />;
  }
};

const Setting = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const [activeKey, setActiveKey] = useState(DEFAULT_TAB);

  const onChangeTab = (key) => {
    setActiveKey(key);
    navigate(`?tab=${key}`, { replace: true });
  };

  useEffect(() => {
    const searchParams = new URLSearchParams(window.location.search);
    const tab = searchParams.get('tab');
    if (tab) {
      // Redirect old tab keys to new ones
      const redirected = TAB_REDIRECT[tab];
      if (redirected) {
        navigate(`?tab=${redirected}`, { replace: true });
        setActiveKey(redirected);
      } else {
        setActiveKey(tab);
      }
    } else {
      onChangeTab(DEFAULT_TAB);
    }
  }, [location.search]);

  if (!isRoot()) {
    return null;
  }

  return (
    <div className="px-2">
      <Layout>
        <Layout.Content>
          <div className="flex flex-col md:flex-row gap-4 mt-2">
            <SettingsSidebar activeKey={activeKey} onChange={onChangeTab} />
            <div className="flex-1 min-w-0">
              {renderContent(activeKey)}
            </div>
          </div>
        </Layout.Content>
      </Layout>
    </div>
  );
};

export default Setting;
