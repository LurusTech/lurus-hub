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

import React, { useEffect, useState, useContext, useRef } from 'react';
import {
  API,
  showError,
  showInfo,
  showSuccess,
  renderQuota,
  copy,
  getQuotaPerUnit,
  isV2Mode,
  v2Url,
} from '../../helpers';
import { Modal } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { UserContext } from '../../context/User';
import { StatusContext } from '../../context/Status';

import RechargeCard from './RechargeCard';
import InvitationCard from './InvitationCard';
import TransferModal from './modals/TransferModal';

const TopUp = () => {
  const { t } = useTranslation();
  const [userState, userDispatch] = useContext(UserContext);
  const [statusState] = useContext(StatusContext);

  const [redemptionCode, setRedemptionCode] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [topUpLink, setTopUpLink] = useState(
    statusState?.status?.top_up_link || '',
  );

  const affFetchedRef = useRef(false);

  // Invitation states
  const [affLink, setAffLink] = useState('');
  const [openTransfer, setOpenTransfer] = useState(false);
  const [transferAmount, setTransferAmount] = useState(0);

  const topUp = async () => {
    if (redemptionCode === '') {
      showInfo(t('请输入兑换码！'));
      return;
    }
    setIsSubmitting(true);
    try {
      const url = isV2Mode() ? v2Url('/redemptions/redeem') : '/api/user/topup';
      const res = await API.post(url, { key: redemptionCode });
      const { success, message, data } = res.data;
      if (success) {
        showSuccess(t('兑换成功！'));
        Modal.success({
          title: t('兑换成功！'),
          content: t('成功兑换额度：') + renderQuota(data),
          centered: true,
        });
        if (userState.user) {
          const updatedUser = {
            ...userState.user,
            quota: userState.user.quota + data,
          };
          userDispatch({ type: 'login', payload: updatedUser });
        }
        setRedemptionCode('');
      } else {
        showError(message);
      }
    } catch (err) {
      showError(t('请求失败'));
    } finally {
      setIsSubmitting(false);
    }
  };

  const openTopUpLink = () => {
    if (!topUpLink) {
      showError(t('超级管理员未设置充值链接！'));
      return;
    }
    window.open(topUpLink, '_blank');
  };

  const getUserQuota = async () => {
    const url = isV2Mode() ? v2Url('/user/me') : '/api/user/self';
    const res = await API.get(url);
    const { success, message, data } = res.data;
    if (success) {
      userDispatch({ type: 'login', payload: data });
    } else {
      showError(message);
    }
  };

  const getAffLink = async () => {
    const res = await API.get('/api/user/aff');
    const { success, message, data } = res.data;
    if (success) {
      const link = `${window.location.origin}/register?aff=${data}`;
      setAffLink(link);
    } else {
      showError(message);
    }
  };

  const transfer = async () => {
    if (transferAmount < getQuotaPerUnit()) {
      showError(t('划转金额最低为') + ' ' + renderQuota(getQuotaPerUnit()));
      return;
    }
    const res = await API.post('/api/user/aff_transfer', {
      quota: transferAmount,
    });
    const { success, message } = res.data;
    if (success) {
      showSuccess(message);
      setOpenTransfer(false);
      getUserQuota();
    } else {
      showError(message);
    }
  };

  const handleAffLinkClick = async () => {
    await copy(affLink);
    showSuccess(t('邀请链接已复制到剪切板'));
  };

  const handleTransferCancel = () => {
    setOpenTransfer(false);
  };

  useEffect(() => {
    if (!userState?.user?.id) {
      getUserQuota();
    }
    setTransferAmount(getQuotaPerUnit());
  }, []);

  useEffect(() => {
    if (affFetchedRef.current) return;
    affFetchedRef.current = true;
    getAffLink();
  }, []);

  useEffect(() => {
    if (statusState?.status) {
      setTopUpLink(statusState.status.top_up_link || '');
    }
  }, [statusState?.status]);

  return (
    <div className='w-full max-w-7xl mx-auto relative min-h-screen lg:min-h-0 mt-[60px] px-2'>
      <TransferModal
        t={t}
        openTransfer={openTransfer}
        transfer={transfer}
        handleTransferCancel={handleTransferCancel}
        userState={userState}
        renderQuota={renderQuota}
        getQuotaPerUnit={getQuotaPerUnit}
        transferAmount={transferAmount}
        setTransferAmount={setTransferAmount}
      />

      <div className='space-y-6'>
        <div className='grid grid-cols-1 lg:grid-cols-12 gap-6'>
          <div className='lg:col-span-7 space-y-6 w-full'>
            <RechargeCard
              t={t}
              redemptionCode={redemptionCode}
              setRedemptionCode={setRedemptionCode}
              topUp={topUp}
              isSubmitting={isSubmitting}
              topUpLink={topUpLink}
              openTopUpLink={openTopUpLink}
              userState={userState}
              renderQuota={renderQuota}
            />
          </div>
          <div className='lg:col-span-5'>
            <InvitationCard
              t={t}
              userState={userState}
              renderQuota={renderQuota}
              setOpenTransfer={setOpenTransfer}
              affLink={affLink}
              handleAffLinkClick={handleAffLinkClick}
            />
          </div>
        </div>
      </div>
    </div>
  );
};

export default TopUp;
