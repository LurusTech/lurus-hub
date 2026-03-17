import { useEffect, useState } from 'react';
import { useParams, useSearchParams } from 'react-router-dom';
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

    // Show fallback option after 3 seconds
    const timer = setTimeout(() => setShowFallback(true), 3000);

    window.location.href = url;

    return () => clearTimeout(timer);
  }, [register, routeSlug, searchParams]);

  return (
    <div className='flex flex-col items-center justify-center min-h-screen bg-gray-50'>
      <Loading />
      {showFallback && (
        <div className='mt-8'>
          <Card className='p-6 shadow-lg'>
            <Typography.Text className='text-gray-600 block text-center'>
              正在跳转到统一登录，请稍候...
            </Typography.Text>
          </Card>
        </div>
      )}
    </div>
  );
};

export default ZitadelRedirect;
