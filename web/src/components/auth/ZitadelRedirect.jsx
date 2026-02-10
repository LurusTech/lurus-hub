import { useEffect, useState } from 'react';
import { useParams, useSearchParams, Link } from 'react-router-dom';
import { getTenantSlug } from '../../helpers';
import Loading from '../common/ui/Loading';
import { Card, Typography } from '@douyinfe/semi-ui';

const DEFAULT_TENANT = 'lurus';

const ZitadelRedirect = ({ register = false }) => {
  const { tenantSlug: routeSlug } = useParams();
  const [searchParams] = useSearchParams();
  const [showFallback, setShowFallback] = useState(false);

  useEffect(() => {
    // Priority: route param > query param > localStorage > default
    const slug =
      routeSlug ||
      searchParams.get('tenant') ||
      getTenantSlug() ||
      DEFAULT_TENANT;

    let url = `/api/v2/${slug}/auth/login?redirect_url=/oauth/zitadel`;
    if (register) {
      url += '&register=true';
    }

    // Show fallback option after 1 second
    const timer = setTimeout(() => setShowFallback(true), 1000);

    window.location.href = url;

    return () => clearTimeout(timer);
  }, [register, routeSlug, searchParams]);

  return (
    <div className='flex flex-col items-center justify-center min-h-screen bg-gray-50'>
      <Loading />
      {showFallback && (
        <div className='mt-8'>
          <Card className='p-6 shadow-lg'>
            <Typography.Text className='text-gray-600 mb-4 block text-center'>
              正在跳转到统一登录...
            </Typography.Text>
            <Link
              to='/login/password'
              className='text-blue-600 hover:text-blue-800 underline block text-center'
            >
              使用密码登录（管理员）
            </Link>
          </Card>
        </div>
      )}
    </div>
  );
};

export default ZitadelRedirect;
