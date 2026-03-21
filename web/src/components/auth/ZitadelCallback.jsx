import React, { useContext, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import {
  API,
  showError,
  showSuccess,
  updateAPI,
  setUserData,
  setTenantSlug,
} from '../../helpers';
import { UserContext } from '../../context/User';
import Loading from '../common/ui/Loading';

const ZitadelCallback = () => {
  const { t } = useTranslation();
  const [, userDispatch] = useContext(UserContext);
  const navigate = useNavigate();
  const [error, setError] = useState(null);

  useEffect(() => {
    let cancelled = false;

    const loadSession = async () => {
      try {
        const res = await API.get('/api/v2/auth/session-info', {
          skipErrorHandler: true,
        });
        if (cancelled) return;

        const { success, message, data } = res.data;
        if (!success) {
          throw new Error(message || t('登录失败'));
        }

        // Store tenant context for V2 mode
        if (data.tenant_slug) {
          setTenantSlug(data.tenant_slug);
        }

        userDispatch({ type: 'login', payload: data });
        localStorage.setItem('user', JSON.stringify(data));
        setUserData(data);
        updateAPI();
        showSuccess(t('登录成功！'));
        navigate('/console');
      } catch (err) {
        if (cancelled) return;
        const msg =
          err?.response?.data?.message || err.message || t('登录失败');
        setError(msg);
        showError(msg);
        setTimeout(() => navigate('/login'), 3000);
      }
    };

    loadSession();

    return () => {
      cancelled = true;
    };
  }, []);

  if (error) {
    return (
      <div style={{ textAlign: 'center', marginTop: '20vh' }}>
        <p>{error}</p>
        <p>{t('正在返回登录页...')}</p>
      </div>
    );
  }

  return <Loading />;
};

export default ZitadelCallback;
