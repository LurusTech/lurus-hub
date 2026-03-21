/**
 * Usage Gauge Component
 * Displays quota usage with progress bar and projected exhaustion.
 *
 * @module components/dashboard/UsageGauge
 */

import React from 'react';
import { Card, Progress, Button, Skeleton } from '@douyinfe/semi-ui';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { renderQuota } from '../../helpers/render';

const LEVEL_COLORS = {
  green: 'var(--semi-color-success)',
  yellow: 'var(--semi-color-warning)',
  red: 'var(--semi-color-danger)',
  critical: '#dc2626',
};

const CARD_STYLE = {
  shadows: '',
  bordered: true,
  headerLine: true,
};

export default function UsageGauge({
  gauge,
  loading,
  CARD_PROPS = CARD_STYLE,
}) {
  const { t } = useTranslation();
  const navigate = useNavigate();

  // Show skeleton while loading (if gauge data is not yet available)
  if (loading && (!gauge || gauge.totalQuota === 0)) {
    return (
      <div className='mb-4'>
        <Card
          {...CARD_PROPS}
          className='!rounded-2xl'
          title={
            <div className='flex items-center gap-2'>
              <span>{t('额度用量')}</span>
            </div>
          }
        >
          <div className='space-y-3'>
            <Skeleton.Paragraph
              active
              rows={1}
              style={{ width: '60%', height: '16px' }}
            />
            <Skeleton.Paragraph
              active
              rows={1}
              style={{ width: '100%', height: '8px', marginTop: '8px' }}
            />
            <Skeleton.Paragraph
              active
              rows={1}
              style={{ width: '80%', height: '14px', marginTop: '8px' }}
            />
          </div>
        </Card>
      </div>
    );
  }

  if (!gauge || gauge.totalQuota === 0) return null;

  const color = LEVEL_COLORS[gauge.level] || LEVEL_COLORS.green;

  return (
    <div className='mb-4'>
      <Card
        {...CARD_PROPS}
        className='!rounded-2xl'
        title={
          <div className='flex items-center gap-2'>
            <span>{t('额度用量')}</span>
          </div>
        }
        headerExtraContent={
          <Button
            theme='solid'
            type='warning'
            size='small'
            onClick={() => navigate('/console/topup')}
          >
            {t('立即充值')}
          </Button>
        }
      >
        <div className='space-y-3'>
          {/* Progress bar */}
          <div>
            <div className='flex justify-between items-center mb-1'>
              <span className='text-sm text-gray-500'>
                {t('已使用')} {renderQuota(gauge.usedQuota)} /{' '}
                {renderQuota(gauge.totalQuota)}
              </span>
              <span
                className='text-sm font-mono font-semibold'
                style={{ color }}
              >
                {gauge.usagePercent}%
              </span>
            </div>
            <Progress
              percent={gauge.usagePercent}
              showInfo={false}
              stroke={color}
              size='large'
              style={{ height: 8 }}
            />
          </div>

          {/* Stats row */}
          <div className='flex flex-wrap gap-4 text-sm'>
            <div>
              <span className='text-gray-500'>{t('剩余额度')}: </span>
              <span className='font-mono font-semibold'>
                {renderQuota(gauge.quota)}
              </span>
            </div>
            {gauge.dailyRate > 0 && (
              <div>
                <span className='text-gray-500'>{t('日均消耗')}: </span>
                <span className='font-mono'>
                  {renderQuota(gauge.dailyRate)}
                </span>
              </div>
            )}
            {gauge.daysRemaining !== null && (
              <div>
                <span className='text-gray-500'>{t('预计用完日期')}: </span>
                <span className='font-mono'>
                  {gauge.exhaustionDate} ({gauge.daysRemaining} {t('天')})
                </span>
              </div>
            )}
          </div>
        </div>
      </Card>
    </div>
  );
}
