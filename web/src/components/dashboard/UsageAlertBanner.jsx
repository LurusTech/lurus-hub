/**
 * Usage Alert Banner Component
 * Shows warning/critical banners when quota is running low.
 *
 * @module components/dashboard/UsageAlertBanner
 */

import React, { useState } from 'react';
import { Banner, Button } from '@douyinfe/semi-ui';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';

export default function UsageAlertBanner({ gauge }) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [dismissed, setDismissed] = useState(false);

  if (!gauge || gauge.level === 'green') return null;

  // Critical banner cannot be dismissed
  const closable = gauge.level !== 'critical';
  if (dismissed && closable) return null;

  const topupButton = (
    <Button
      theme='solid'
      type={gauge.level === 'critical' ? 'danger' : 'warning'}
      size='small'
      onClick={() => navigate('/console/topup')}
    >
      {gauge.level === 'critical' ? t('立即充值') : t('充值')}
    </Button>
  );

  if (gauge.level === 'critical') {
    return (
      <div className='mb-4' role='alert' aria-live='assertive'>
        <Banner
          type='danger'
          fullMode={false}
          closeIcon={null}
          description={
            <div className='flex items-center justify-between flex-wrap gap-2'>
              <span>
                {t('额度即将耗尽')} ({gauge.usagePercent}%)
                {gauge.daysRemaining !== null &&
                  ` - ${t('预计')} ${gauge.daysRemaining} ${t('天后用完')}`}
              </span>
              {topupButton}
            </div>
          }
        />
      </div>
    );
  }

  if (gauge.level === 'red') {
    return (
      <div className='mb-4' role='alert' aria-live='assertive'>
        <Banner
          type='danger'
          fullMode={false}
          onClose={() => setDismissed(true)}
          description={
            <div className='flex items-center justify-between flex-wrap gap-2'>
              <span>
                {t('额度即将耗尽')} ({gauge.usagePercent}%)
                {gauge.daysRemaining !== null &&
                  ` - ${t('预计')} ${gauge.daysRemaining} ${t('天后用完')}`}
              </span>
              {topupButton}
            </div>
          }
        />
      </div>
    );
  }

  // Yellow
  return (
    <div className='mb-4' role='alert' aria-live='polite'>
      <Banner
        type='warning'
        fullMode={false}
        onClose={() => setDismissed(true)}
        description={
          <div className='flex items-center justify-between flex-wrap gap-2'>
            <span>
              {t('额度已使用')} {gauge.usagePercent}%
              {gauge.daysRemaining !== null &&
                `，${t('预计')} ${gauge.daysRemaining} ${t('天后用完')}`}
            </span>
            {topupButton}
          </div>
        }
      />
    </div>
  );
}
